package cmd

import (
	"fmt"
	"os"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/spf13/cobra"
)

// Declare the flag variable at package level so runDiff can read it
var policiesFile string
var subscriptionID string

// init() is called automatically by Go before main().
// This is the standard Cobra pattern for registering flags.
func init() {
	// Register --policies_dir flag on diffCmd (not rootCmd)
	diffCmd.Flags().StringVar(
		&policiesFile,                        // where to store the value
		"policiesFile",                       // flag name
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

	remotePolicy, err := getRemotePolicy(policiesFile)
	if err != nil {
		return fmt.Errorf("Authentication failed: %w", err)
	}
	fmt.Println(remotePolicy)

	// 2. Get the rawPolicy
	policy, err := engine.LoadPolicy(policiesFile)
	if err != nil {
		return fmt.Errorf("Could not load policy. Check the path: %s, %w", policiesFile, err)
	}
	fmt.Println(policy)

	return nil

}

func getRemotePolicy(policyFile string) (string, error) {
	// Authenticate
	cred, err := engine.Authenticate()
	if err != nil {
		return "", err
	}

	policy, err := engine.LoadPolicy(policyFile)
	if err != nil {
		return "", err
	}

	// Get the azureBackupPolicy
	azureBackupPolicy, err := engine.GetAzureBackupPolicy(cred, subscriptionID, policy.Spec.StorageProvider.Resource_group, policy.Spec.StorageProvider.Vault, policy.Spec.StorageProvider.BackupPolicy)
	if err != nil {
		return "", fmt.Errorf("Could get the backup policy from Azure. Check the file: %s\n Error: %w", policiesFile, err)
	}

	return azureBackupPolicy, nil

}
