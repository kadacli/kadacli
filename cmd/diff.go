package cmd

import (
	"fmt"
	"os"
	"reflect"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local backup policies against live Azure configuration",
	Long: `Loads backup policy YAML files from a directory and compares each
policy's desired state against the currently configured Azure backup policy.`,
	RunE: runDiff,
}

var green = color.New(color.FgGreen).SprintFunc()
var cyan = color.New(color.FgHiCyan).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()

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
	var validMaps []map[string]interface{}
	var diffMaps []map[string]interface{}
	var completeMaps []map[string]interface{}
	var overwriteMaps []map[string]interface{}

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

		equalSettings, diff, complete, overwrite, err := engine.DiffPolicy(localPolicies[i], remoteVault)
		if err != nil {
			errs = append(errs, err)
		}

		if !applying {
			fmt.Printf("\n------------ Policy: %s ----------------", localPolicies[i].Metadata.Name)

			if len(diff) > 0 {
				fmt.Printf("\nCHANGES:\n")
				for key, value := range overwrite {
					from := value
					if isZero(value) {
						from = "(not set)"
					}
					fmt.Printf("%v:		%v=>%v\n", key, from, diff[key])
				}
			} else {
				fmt.Println(green("NO CHANGES"))
			}

			if len(equalSettings) > 0 {
				fmt.Printf("\nNOT CHANGED:\n")
				for key, value := range equalSettings {
					from := value
					if isZero(value) {
						from = "(not set)"
					}
					fmt.Printf("  %s: %v\n", key, from)
				}
			}

		}

		validMaps = append(validMaps, equalSettings)
		diffMaps = append(diffMaps, diff)
		completeMaps = append(completeMaps, complete)
		overwriteMaps = append(overwriteMaps, overwrite)

	}

	return validMaps, diffMaps, completeMaps, overwriteMaps, errs
}

func isZero(v any) bool {
	if v == nil {
		return true
	}
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}
