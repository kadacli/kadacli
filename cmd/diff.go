package cmd

import (
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Check staging differences",
	Run:   runDiff,
}

func runDiff(cmd *cobra.Command, args []string) {
	// TODO: Write runDiff.
	// LBH
}
