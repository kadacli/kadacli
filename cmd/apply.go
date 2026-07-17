package cmd

import (
	"fmt"
	"os"
	"strings"

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
		var resourceGroup, resourceName, resourceType string

		switch payload := rt.Payload.(type) {
		case azure.ResolvedVM:
			resourceGroup, resourceName, resourceType = payload.ResourceGroup, payload.ResourceName, payload.Resource.Type
			n, messages, err = azure_engine.ApplyVM(cmd.Context(), cred, payload, dryRun)
		case azure.ResolvedDisk:
			resourceGroup, resourceName, resourceType = payload.ResourceGroup, payload.ResourceName, payload.Resource.Type
			n, messages, err = azure_engine.ApplyDisk(cmd.Context(), cred, payload, dryRun)
		case azure.ResolvedSqlDb:
			resourceGroup, resourceName, resourceType = payload.ResourceGroup, payload.ResourceName, payload.Resource.Type
			n, messages, err = azure_engine.ApplySqlDb(cmd.Context(), cred, payload, dryRun)
		default:
			applyErrors = append(applyErrors, fmt.Errorf("provider %q is not yet supported", rt.Provider))
			continue
		}

		fmt.Println(colorCyan(fmt.Sprintf("\n──── %s/%s (%s) ────", resourceGroup, resourceName, resourceType)))
		for _, msg := range messages {
			fmt.Println("  " + colorizeApplyMessage(msg))
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

// colorizeApplyMessage colors a message returned from an Apply* function based
// on its content: "✔ " prefix means a change was made, "[dry-run]" means a
// dry-run preview, and anything else is an informational/pending note.
func colorizeApplyMessage(msg string) string {
	switch {
	case strings.HasPrefix(msg, "✔"):
		return colorGreen(msg)
	case strings.HasPrefix(msg, "[dry-run]"):
		return colorYellow(msg)
	default:
		return colorYellow(msg)
	}
}
