package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate policy YAML files without connecting to any provider",
	Run:   runValidate,
}

func runValidate(cmd *cobra.Command, args []string) {
	dir := viper.GetString("policies_dir")

	// Load all YAML files from the policies directory
	raw, err := engine.LoadAllPolicies(dir)
	if err != nil {
		fmt.Println(err)
	}

	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgHiCyan).SprintFunc()

	// Resolve inheritance from _defaults/
	policies, errs := engine.Resolve(raw)
	if len(errs) > 0 {
		for i := range errs {
			fmt.Println(red(fmt.Sprintf("✖ %s", errs[i])))
		}
		os.Exit(1)
	}
	if len(policies) > 0 {
		for _, policy := range policies {
			fmt.Println(green(fmt.Sprintf("✔ Policy %s is valid!", policy.Metadata.Name)))
		}
	}
	fmt.Println(cyan("All your policies have been checked!"))
}
