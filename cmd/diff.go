package cmd

import (
	"fmt"
	"os"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/spf13/cobra"
)

// Declare the flag variable at package level so runDiff can read it
var policyFile string
var subscriptionID string

func init() {
	// Register --policies_dir flag on diffCmd (not rootCmd)
	diffCmd.Flags().StringVar(
		&policyFile,                          // where to store the value
		"policyFile",                         // flag name
		"yaml_files/valid_yaml/staging.yaml", // default value
		"Directory containing backup policy YAML files",
	)

	// In diff.go init()
	diffCmd.Flags().StringVar(
		&subscriptionID,
		"subscription",
		os.Getenv("AZURE_SUBSCRIPTION_ID"),
		"Azure subscription ID",
	)

	// Register diffCmd as a subcommand of rootCmd
	rootCmd.AddCommand(diffCmd)
}

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
	// TODO: Write runDiff.
	// LBH

	if subscriptionID == "" {
		return fmt.Errorf(
			"subscription ID is required: set AZURE_SUBSCRIPTION_ID as an environment variable or use the --subscription flag \n")
	}

	// 1. Get the rawPolicy
	localPolicy, err := engine.LoadPolicy(policyFile)
	if err != nil {
		return fmt.Errorf("Could not load policy. Check the path: %s, %w", policyFile, err)
	}
	errs := engine.ValidatePolicy(localPolicy)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Println(err)
		}
		return fmt.Errorf("The provided yaml file is not valid. Check: %s", policyFile)
	}

	validLocalPolicy := models.BackupPolicy(localPolicy)

	remoteVault, err := engine.GetBackupPolicyVM(
		validLocalPolicy.Spec.StorageProvider.Resource_group,
		validLocalPolicy.Spec.StorageProvider.Vault,
		validLocalPolicy.Spec.StorageProvider.BackupPolicy,
		validLocalPolicy.Spec.StorageProvider.Provider,
		validLocalPolicy.Spec.Targets[0].Selectors[0].ResourceName,
		subscriptionID)
	if err != nil {
		return fmt.Errorf("Failed to get the remote policy: %w", err)
	}

	valid, diff, err := engine.DiffPolicy(validLocalPolicy, remoteVault)
	if err != nil {
		return fmt.Errorf("Error when comparing the policies: %s and %s", policyFile, *remoteVault.Name)
	}

	fmt.Println(valid)
	fmt.Println(diff)

	return err

}
