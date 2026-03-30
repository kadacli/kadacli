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
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("policies_dir")

	// Load all YAML files from the policies directory
	raw, err := engine.LoadAllPolicies(dir)
	if err != nil {
		return err
	}

	// Resolve inheritance from _defaults/
	policies, err := engine.Resolve(raw)
	if err != nil {
		return err
	}

	// Validate each resolved policy
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	failed := 0

	for _, p := range policies {
		if err := engine.ValidatePolicy(p); err != nil {
			fmt.Printf("  %s  %s\n     %s\n", red("✘"), p.SourcePath, err)
			failed++
		} else {
			fmt.Printf("  %s  %s\n", green("✔"), p.Metadata.Name)
		}
	}

	// Exit 1 if any policy failed — CI gates on this
	if failed > 0 {
		fmt.Printf("\n%d of %d policies failed validation.\n", failed, len(policies))
		os.Exit(1)
	}

	fmt.Printf("\n%d %s validated successfully.\n", len(policies), pluralise("policy", "policies", len(policies)))
	return nil
}

func pluralise(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}
