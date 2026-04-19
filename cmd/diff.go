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
		"policies_dir",                       // flag name
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
	RunE: runDiff, // RunE lets you return an error, which Cobra prints cleanly
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

	// 1. Authenticate
	cred, err := engine.Authenticate()
	if err != nil {
		return fmt.Errorf("Authentication failed: %w", err)
	}

	// 2. Get the rawPolicy
	policy, err := engine.LoadPolicy(policiesFile)
	if err != nil {
		return fmt.Errorf("Could not load policy. Check the path: %s, %w", policiesFile, err)
	}

	// 3. Get the vault name and resource group
	vaultName := policy.Spec.StorageProvider.Vault
	resourceGroup := policy.Spec.StorageProvider.Resource_group

	azureVault, err := engine.GetAzureVault(cred, subscriptionID, resourceGroup, vaultName)
	if err != nil {
		return fmt.Errorf("Could not get azure vault. Check the file: %s, %w", policiesFile, err)
	}
	// I now have the azureVault. Now I have to get the relevant settings

	fmt.Println(azureVault)

	return nil

}
