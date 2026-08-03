package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// annotationExitCodes marks the commands that end in a 0/1/2 compliance
// verdict, so Execute knows which ones should print a success line.
const annotationExitCodes = "kada_exit_codes"

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
	cmd, err := rootCmd.ExecuteC()

	if err == nil {
		if reportsExitCode(cmd) {
			fmt.Println(colorGreen("\nExit code 0 — compliant"))
		}
		return
	}

	// Commands that reached a verdict say so by returning an *exitError.
	var ee *exitError
	if errors.As(err, &ee) {
		if ee.code == exitDrift {
			fmt.Fprintln(os.Stderr, colorYellow("\nExit code 1 — "+ee.Error()))
		} else {
			fmt.Fprintln(os.Stderr, colorRed("\nExit code 2 — "+ee.Error()))
		}
		os.Exit(ee.code)
	}

	// Everything else — usage errors, unknown flags, unknown commands — is a
	// failure. It must never fall through to a 0 exit: a typo'd flag that
	// exits 0 passes a pipeline green having done nothing.
	fmt.Fprintln(os.Stderr, colorRedBold("Error: "+err.Error()))
	os.Exit(exitFailure)
}

// reportsExitCode reports whether cmd is one of the commands that ends with an
// explicit compliance verdict. init and the help output make no such claim.
func reportsExitCode(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Annotations[annotationExitCodes] != "true" {
		return false
	}
	// --help prints usage and succeeds; that is not a statement about the
	// state of anyone's backups.
	if help, err := cmd.Flags().GetBool("help"); err == nil && help {
		return false
	}
	return true
}

func init() {
	rootCmd.PersistentFlags().String("dir", ".", "Directory containing Policy and Assignment YAML files")

	viper.BindPFlag("dir", rootCmd.PersistentFlags().Lookup("dir"))

	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(initCmd)
}
