package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
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

	subscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionId == "" {
		subscriptionId = viper.GetString("subscription")
	}
	if subscriptionId == "" {
		return fmt.Errorf(
			"subscription ID is required: set AZURE_SUBSCRIPTION_ID as an environment variable or use the --subscription flag \n")
	}

	errs := runDiff(dir, subscriptionId)
	if len(errs) > 0 {
		for i := range errs {
			fmt.Printf("\nERROR: %v", errs[i])
		}
	}

	return nil
}

func runDiff(path string, subscriptionId string) []error {
	var errs []error

	// get all policyfiles
	rawPolicies, errs := engine.LoadAllPolicies(path)
	if len(rawPolicies) == 0 {
		return errs
	}

	localPolicies, errs := engine.Resolve(rawPolicies)
	if len(localPolicies) == 0 {
		return errs
	}

	for i := range localPolicies {
		// Connect to Azure and get the active policy
		remoteVault, err := engine.GetBackupPolicyVM(
			localPolicies[i].Spec.StorageProvider.Resource_group,
			localPolicies[i].Spec.StorageProvider.Vault,
			localPolicies[i].Spec.StorageProvider.BackupPolicy,
			localPolicies[i].Spec.StorageProvider.Provider,
			localPolicies[i].Spec.Targets[0].Selectors[0].ResourceName,
			subscriptionId)
		if err != nil {
			errs = append(errs, err)
		}

		valid, diff, complete, overwrite, err := engine.DiffPolicy(localPolicies[i], remoteVault)
		if err != nil {
			errs = append(errs, err)
		}

		fmt.Printf("The following settings will be changed in the policy: %s\n", localPolicies[i].Metadata.Name)
		printMap(diff)
		fmt.Printf("\nThe settings will be changed to: \n")
		printMap(overwrite)

		fmt.Printf("\nThe following settings will not be changed: \n")
		printMap(valid)
		fmt.Println("\nThe final settings will be: ")
		printMap(complete)

	}
	return errs
}

func printMap(m map[string]any) {
	for k, v := range m {
		fmt.Printf("%s: ", k)
		jsonData, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
		}
		fmt.Println(string(jsonData))
	}
}
