package cmd

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply backup policy",
	Run:   runApply,
}

func runApply(cmd *cobra.Command, args []string) {
	// TODO: Write runApply. Should validate & idempotent & apply, using engine/executor.
	// DANBJ
	dir := viper.GetString("policies_dir")
	validate(dir)
	cred, err := engine.Authenticate()
	if err != nil {
		fmt.Printf("Authentication error: %s", err)
		os.Exit(1)
	}
	// TOOD: diff(cred)
	// Run validate
	// Authenticate
	// Run diff
	// Run apply
	apply(cred, nil)
	// Report?
}

func apply(cred *azidentity.DefaultAzureCredential, diff map[string]interface{}) {
	// Extract values from map
	//Apply
	engine.ApplyAzureVM(
		cred,
		os.Getenv("AZURE_SUBSCRIPTION_ID"),
		diff,
	)
}
