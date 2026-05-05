package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "bpac",
	Short: "bpac - declare, version, and apply backup policies across cloud providers using YAML",
	Long: `bpac (Backup Policy as Code) — define Azure backup policies as YAML files,
diff desired state against live Azure state, and apply policies idempotently.

Exit codes: 0 = success/compliant, 1 = violations found, 2 = error`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(2)
	}
}

func init() {
	rootCmd.PersistentFlags().String("dir", ".", "Directory containing Policy and Assignment YAML files")
	rootCmd.PersistentFlags().String("subscription", "", "Azure subscription ID (overrides Assignment metadata)")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("output", "text", "Output format: text | json")

	viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("subscription", rootCmd.PersistentFlags().Lookup("subscription"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(initCmd)

	applyCmd.Flags().Bool("force", false, "Apply without confirmation prompt")
	applyCmd.Flags().Bool("dry-run", false, "Show what would be applied without making changes")
	diffCmd.Flags().Bool("dry-run", false, "Same as diff (no-op flag for pipeline compatibility)")
	auditCmd.Flags().String("subscription", "", "Subscription ID to audit")
	auditCmd.Flags().String("policy", "", "Path to a Policy YAML file to audit against")
}
