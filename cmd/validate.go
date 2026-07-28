package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate policy and assignment YAML files without connecting to vendor",
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	fmt.Println(colorKada("──────────────── kåda — backup policy as code ────────────────"))
	dir := viper.GetString("dir")
	resolved, err := loadAndValidate(dir)
	if err != nil {
		return err
	}
	fmt.Println(colorCyan(fmt.Sprintf("All files valid — %d resolved target(s)", len(resolved))))
	return nil
}

// loadAndValidate loads, validates, and resolves all YAML files in dir.
// Prints per-file results. The caller is responsible for any summary line.
// Used by validate, diff, and apply.
func loadAndValidate(dir string) ([]models.ResolvedTarget, error) {
	// Step 1 — load
	pathPolicies, pathAssignments, loadErrs := engine.LoadAll(dir)
	for _, e := range loadErrs {
		fmt.Fprintln(os.Stderr, colorRed("✖ Load: "+e.Error()))
	}
	if len(loadErrs) > 0 {
		return nil, fmt.Errorf("Loading failed with %d error(s)", len(loadErrs))
	}

	// Step 2 — validate policies
	var validatedPolicies []models.PathPolicy
	failed := false
	for _, pp := range pathPolicies {
		if err := engine.ValidatePolicy(pp.Policy, pp.Path); err != nil {
			fmt.Fprintln(os.Stderr, colorRed("✖ "+err.Error()))
			failed = true
		} else {
			fmt.Println(colorGreen(fmt.Sprintf("✔ Policy %q (%s)", pp.Policy.Metadata.Name, pp.Path)))
			validatedPolicies = append(validatedPolicies, pp)
		}
	}

	// Step 3 — validate assignments
	var validatedAssignments []models.PathAssignment
	for _, pa := range pathAssignments {
		if err := engine.ValidateAssignment(pa.Assignment, pa.Path); err != nil {
			fmt.Fprintln(os.Stderr, colorRed("✖ "+err.Error()))
			failed = true
		} else {
			fmt.Println(colorGreen(fmt.Sprintf("✔ Assignment %q (%s)", pa.Assignment.Metadata.Name, pa.Path)))
			validatedAssignments = append(validatedAssignments, pa)
		}
	}

	if failed {
		return nil, fmt.Errorf("Validation failed")
	}

	// Step 4 — resolve
	resolved, resolveErrs := engine.Resolve(validatedPolicies, validatedAssignments)
	if len(resolveErrs) > 0 {
		for _, e := range resolveErrs {
			fmt.Fprintln(os.Stderr, colorRed("✖ Resolve: "+e.Error()))
		}
		return nil, fmt.Errorf("Resolve failed with %d error(s)", len(resolveErrs))
	}

	return resolved, nil
}
