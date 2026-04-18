package cmd

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices"
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
			"subscription ID is required: set AZURE_SUBSCRIPTION_ID or use --subscription")
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

	var resourceGroups []string
	for _, target := range policy.Spec.Targets {
		for _, selector := range target.Selectors {
			resourceGroups = append(resourceGroups, selector.ResourceGroup)
		}
	}

	var vaultNames []string
	for _, provider := range policy.Spec.StorageProviders {
		vaultNames = append(vaultNames, provider.Vault)
	}
	// get the vaults
	var azVaults *armrecoveryservices.Vault[]
	for _,vaultName := range vaultNames {
		vault, err = engine.GetAzureVault(cred, subscriptionID, )
		azVaults = append(azVaults)
	}
	engine.GetAzureVault(cred, subscriptionID, resourceGroup, vaultName)

	return nil

}
