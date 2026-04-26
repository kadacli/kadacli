package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "backupctl",
	Short: "backupctl - declare, version, and apply backup policies across backup & restore providers using YAML",
	Long: `backupctl - declare, version, and apply backup policies across backup & restore providers using YAML
			
	
	made by backup engineers tired of clicking through the GUI`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error in CLI: '%s'", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("policies-dir", "policies", "Directory containing policy YAML files")
	rootCmd.PersistentFlags().String("subscriptionId", "subscription", "The SubscriptionID to connect to")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")

	viper.BindPFlag("policies_dir", rootCmd.PersistentFlags().Lookup("policies-dir"))
	viper.BindPFlag("subscription", rootCmd.PersistentFlags().Lookup("subscription"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// Add commands under here, for example: rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(diffCmd)

	// Add --force flag to apply command
	applyCmd.Flags().Bool("force", false, "Force apply without confirmation")
}
