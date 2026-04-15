package cmd

import (
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply backup policy",
	Run:   runApply,
}

func runApply(cmd *cobra.Command, args []string) {
	// TODO: Write runApply. Should validate & idempotent & apply, using engine/executor.
	// DANBJ
}
