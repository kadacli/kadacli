package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "backupctl",
	Short: "backupctl - a simple CLI to track your backup policies",
	Long: `backupctl - a simple CLI to track your backup policies
			made by backup engineers tired of clicking through the GUI`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error in CLI: '%s'", err)
		os.Exit(1)
	}
}
