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
	RunE: runDiff,
}

// Takes a subscription ID and a yaml file and compares the
// live azure policy to the yaml file
func runDiff(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("policies_dir")

	_, _, _, _, errs := diff(dir, false)
	if len(errs) > 0 {
		for i := range errs {
			fmt.Printf("\nERROR: %v", errs[i])
		}
	}

	return nil
}

func diff(path string, applying bool) (
	[]map[string]interface{},
	[]map[string]interface{},
	[]map[string]interface{},
	[]map[string]interface{},
	[]error) {
	var errs []error
	var validMaps []map[string]any
	var diffMaps []map[string]any
	var completeMaps []map[string]any
	var overwriteMaps []map[string]any

	subscriptionId := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionId == "" {
		subscriptionId = viper.GetString("subscription")
	}
	if subscriptionId == "" {
		errs = append(errs, fmt.Errorf("Subscription ID not set"))
		return nil, nil, nil, nil, errs
	}

	localPolicies := validate(path, false)
	// Authenticate
	cred, err := engine.Authenticate()
	if err != nil {
		errs = append(errs, err)
		return validMaps, diffMaps, completeMaps, overwriteMaps, errs
	}

	for i := range localPolicies {
		// Connect to Azure and get the active policy
		remoteVault, err := engine.GetBackupPolicyVM(
			cred,
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

		if !applying {
			fmt.Printf("The following settings will be changed in the policy: %s\n", localPolicies[i].Metadata.Name)
			printMap(diff)
			fmt.Printf("\nThe settings will be changed to: \n")
			printMap(overwrite)

			fmt.Printf("\nThe following settings will not be changed: \n")
			printMap(valid)
			fmt.Println("\nThe final settings will be: ")
			printMap(complete)
		}

		validMaps = append(validMaps, valid)
		diffMaps = append(diffMaps, diff)
		completeMaps = append(completeMaps, complete)
		overwriteMaps = append(overwriteMaps, overwrite)

	}

	return validMaps, diffMaps, completeMaps, overwriteMaps, errs
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
