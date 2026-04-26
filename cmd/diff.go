package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local backup policies against live Azure configuration",
	Long: `Loads backup policy YAML files from a directory and compares each
policy's desired state against the currently configured Azure backup policy.`,
	RunE: Diff,
}

// Takes a subscription ID and a yaml file and compares the
// live azure policy to the yaml file
func Diff(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("policies_dir")
	policyFile := dir + "/valid_yaml/staging.yaml" // TODO remove hardcoding to specific file

	subscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionId == "" {
		subscriptionId = viper.GetString("subscription")
	}
	if subscriptionId == "" {
		return fmt.Errorf(
			"subscription ID is required: set AZURE_SUBSCRIPTION_ID as an environment variable or use the --subscription flag \n")
	}

	equalSettings, settingsToApply, finalPolicy, err := runDiff(policyFile, subscriptionId)

	for setting, _ := range equalSettings {
		fmt.Println(setting)
	}

	fmt.Println("\nSettings to apply")
	for setting, detail := range settingsToApply {
		fmt.Printf("%s:\n", setting)
		jsonData, err := json.MarshalIndent(detail, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
		}
		fmt.Println(string(jsonData))
	}
	fmt.Println("\nThe final settings should be:")
	for setting, detail := range finalPolicy {
		fmt.Printf("%s:\n", setting)
		jsonData, err := json.MarshalIndent(detail, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
		}
		fmt.Println(string(jsonData))
	}

	return err

}

func runDiff(policyFile string, subscriptionId string) (map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {
	// 1. Get the rawPolicy
	localPolicy, err := engine.LoadPolicy(policyFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Could not load policy. Check the path: %s, %w", policyFile, err)
	}
	errs := engine.ValidatePolicy(localPolicy)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		return nil, nil, nil, fmt.Errorf("The provided yaml file is not valid. Check: %s", policyFile)
	}
	// Validate the localPolicy
	validLocalPolicy := models.BackupPolicy(localPolicy)

	// Connect to Azure and get the active policy
	remoteVault, err := engine.GetBackupPolicyVM(
		validLocalPolicy.Spec.StorageProvider.Resource_group,
		validLocalPolicy.Spec.StorageProvider.Vault,
		validLocalPolicy.Spec.StorageProvider.BackupPolicy,
		validLocalPolicy.Spec.StorageProvider.Provider,
		validLocalPolicy.Spec.Targets[0].Selectors[0].ResourceName,
		subscriptionId)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to get the remote policy: %w", err)
	}

	valid, diff, complete, err := engine.DiffPolicy(validLocalPolicy, remoteVault)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error when comparing the policies: %s and %s", policyFile, *remoteVault.Name)
	}

	return valid, diff, complete, err

}
