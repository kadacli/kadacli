package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	azure_external "github.com/dbjorklund/backup-policy-as-code/external/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply backup policies idempotently to Azure",
	Long: `Validates policies, diffs desired state against live Azure, and applies
only the resources that are out of desired state. Safe to run repeatedly.`,
	RunE: runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("dir")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	resolved, err := loadAndValidate(dir)
	if err != nil {
		return err
	}

	// TODO: differentiate auth based on vendor
	// TODO: Move auth to internal.engine
	cred, err := azure_external.Authenticate()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	var applyErrors []error
	applied := 0

	for _, rt := range resolved {
		var n int
		var messages []string
		var err error

		switch payload := rt.Payload.(type) {
		case azure.ResolvedVM:
			n, messages, err = azure_engine.ApplyVM(cmd.Context(), cred, payload, dryRun)
		case azure.ResolvedDisk:
			n, messages, err = azure_engine.ApplyDisk(cmd.Context(), cred, payload, dryRun)
		case azure.ResolvedSqlDb:
			n, messages, err = azure_engine.ApplySqlDb(cmd.Context(), cred, payload, dryRun)
		default:
			applyErrors = append(applyErrors, fmt.Errorf("provider %q is not yet supported", rt.Provider))
			continue
		}

		for _, msg := range messages {
			fmt.Println("  " + msg)
		}
		applied += n
		if err != nil {
			applyErrors = append(applyErrors, err)
			fmt.Fprintln(os.Stderr, colorRed("✖ "+err.Error()))
		}
	}

	fmt.Printf("\nApplied %d change(s) with %d error(s)\n", applied, len(applyErrors))
	if len(applyErrors) > 0 {
		return fmt.Errorf("apply completed with %d error(s)", len(applyErrors))
	}
	return nil
}
