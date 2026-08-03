package cmd

import (
	"fmt"
	"os"
	"strings"

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
		if strings.Contains(err.Error(), "1") {
			fmt.Fprintln(os.Stderr, colorYellow("\n"+err.Error()))
			os.Exit(1)
		} else if strings.Contains(err.Error(), "2") {
			fmt.Fprintln(os.Stderr, colorRed("\n"+err.Error()))
			os.Exit(2)
		}
	} else {
		fmt.Println(colorGreen("\nExit Code: 0 - Compliant"))
		os.Exit(0)
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
