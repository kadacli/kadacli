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
	viper.BindPFlag("policies_dir", rootCmd.PersistentFlags().Lookup("policies-dir"))

	// Add commands under here, for example: rootCmd.AddCommand(initCmd)
}
