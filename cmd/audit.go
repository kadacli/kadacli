package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Scan a subscription and report compliant, drifted, and unprotected resources",
	Long: `Scans an Azure subscription and reports each resource's backup status
against a policy definition. Resources are classified as:

  Compliant   — backup configured and matches desired policy
  Drifted     — backup configured but settings differ from desired policy
  Unprotected — no backup configured

Exit codes: 0 = all compliant, 1 = violations found, 2 = error`,
	RunE: runAudit,
}

func runAudit(cmd *cobra.Command, args []string) error {
	// TODO: implement for 1.0.0
	// Steps:
	//   1. Load policy files from --policy flag
	//   2. Authenticate to Azure
	//   3. Scan subscription with audit.Scanner
	//   4. Detect drift with audit.DetectDrift
	//   5. Find unprotected with audit.FindUnprotected
	//   6. Build report with audit.BuildReport
	//   7. Output via audit.FormatText or audit.FormatJSON
	return fmt.Errorf("audit command not yet implemented")
}
