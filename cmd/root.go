package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "kada",
	Short: "kada - declare, version, and apply backup policies across cloud providers using YAML",
	Long: `kada — define backup policies as YAML files,
diff desired state against live state, and apply policies idempotently.

Exit codes: 0 = success/compliant, 1 = violations found, 2 = error`,
	SilenceUsage:  true, // don't print usage on error
	SilenceErrors: true, // we print errors ourselves in Execute
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(2)
	}
}

func init() {
	rootCmd.PersistentFlags().String("dir", ".", "Directory containing Policy and Assignment YAML files")

	viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir"))

	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(initCmd)
}
