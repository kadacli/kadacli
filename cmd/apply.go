package cmd

import (
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Validate and apply backup policy",
	Run:   runApply,
}

func runApply(cmd *cobra.Command, args []string) {
	// TODO: Write runApply. Should validate & idempotent & apply, using engine/executor.
	// LBH
}
