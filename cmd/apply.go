package cmd

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/fatih/color"
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

	// Set colors:
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	dir := viper.GetString("policies_dir")

	validate(dir, true)
	cred, err := engine.Authenticate()

	if err != nil {
		fmt.Printf("Authentication error: %s", err)
		os.Exit(1)
	}

	_, settingsToApply, _, _, errs := diff(dir, true)
	if len(errs) > 0 {
		for i := range errs {
			fmt.Println(red("\nERROR: %v", errs[i]))
		}
	}

	for i := range settingsToApply {
		err := apply(cred, settingsToApply[i])
		if err != nil {
			fmt.Println(red(fmt.Sprintf("✖ Error when applying policy: %s for resource %s", err, settingsToApply[i]["resourceName"])))
		} else {
			fmt.Println(green(fmt.Sprintf("✔ Successfully applied policy for: %s", settingsToApply[i]["resourceName"])))
		}
	}
	// apply "settingsToApply" in a loop

	// TOOD: diff(cred)
	// Run validate
	// Authenticate
	// Run diff
	// Run apply
	//apply(cred, nil)
	// Report.
}

func apply(cred *azidentity.DefaultAzureCredential, settings map[string]interface{}) error {
	// Extract values from settings app map
	//Apply
	err := engine.ApplyAzureVM(
		cred,
		os.Getenv("AZURE_SUBSCRIPTION_ID"),
		settings["resourceGroup"].(string),
		settings["resourceName"].(string),
		settings["vaultName"].(string),
		settings["policyName"].(string),
		settings["diskLunList"].([]int32),
		settings["isInclusionList"].(bool))

	if err != nil {
		return fmt.Errorf("Something went wrong: %v", err)
	}
	return nil
}
