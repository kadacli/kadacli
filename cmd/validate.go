package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

var validateCmd = &cobra.Command{
	Use:         "validate",
	Short:       "Validate policy and assignment YAML files without connecting to vendor",
	Args:        cobra.NoArgs,
	RunE:        runValidate,
	Annotations: map[string]string{annotationExitCodes: "true"},
}

func runValidate(cmd *cobra.Command, args []string) error {
	fmt.Println(colorKadaBold("──────────────── kåda — backup policy as code ────────────────"))
	dir := viper.GetString("dir")
	resolved, err := loadAndValidate(dir)
	if err != nil {
		return failure(err)
	}
	fmt.Println(colorKada(fmt.Sprintf("All files valid — %d resolved target(s)", len(resolved))))
	return nil
}

// formatFileError renders a failed file the same way a successful one is
// rendered — ✖ Policy "name" (path): reason — so both line up in the output.
// Validators prefix their messages with the source path, so it is stripped
// here to avoid printing the path twice; multi-line detail (one bullet per
// failed field) is indented under the header.
func formatFileError(kind, name, path string, err error) string {
	msg := strings.TrimSpace(err.Error())
	msg = strings.TrimSpace(strings.TrimPrefix(msg, path+":"))
	for _, preamble := range []string{"Validation failed:", "Assignment validation failed:"} {
		msg = strings.TrimSpace(strings.TrimPrefix(msg, preamble))
	}

	if name == "" {
		name = "<unnamed>"
	}
	header := fmt.Sprintf("✖ %s %q (%s):", kind, name, path)

	lines := strings.Split(msg, "\n")
	if len(lines) == 1 {
		// A single failed field needs no bullet — it fits on the header line.
		return header + " " + strings.TrimPrefix(lines[0], "- ")
	}
	for i, line := range lines {
		lines[i] = "    " + strings.TrimSpace(line)
	}
	return header + "\n" + strings.Join(lines, "\n")
}

// loadAndValidate loads, validates, and resolves all YAML files in dir.
// Prints per-file results. The caller is responsible for any summary line.
// Used by validate, diff, and apply.
func loadAndValidate(dir string) ([]models.ResolvedTarget, error) {
	// Step 1 — load
	loaded := engine.LoadAll(dir)
	pathPolicies, pathAssignments := loaded.Policies, loaded.Assignments

	for _, e := range loaded.Errs {
		fmt.Fprintln(os.Stderr, colorRed("✖ Load: "+e.Error()))
	}
	if len(loaded.Errs) > 0 {
		return nil, fmt.Errorf("Loading failed with %d error(s)", len(loaded.Errs))
	}

	if len(loaded.Skipped) > 0 {
		fmt.Println(colorGray(fmt.Sprintf("Skipped %d YAML file(s) that are not kåda documents", len(loaded.Skipped))))
	}

	// A scan that read nothing must not report success. Every command here
	// answers a question about a set of policies, and the honest answer over an
	// empty set is "I did not find any", not "compliant" — a wrong --dir or a
	// moved policy directory would otherwise keep a pipeline green forever
	// while checking nothing at all.
	if len(pathPolicies) == 0 && len(pathAssignments) == 0 {
		return nil, fmt.Errorf(
			"No Policy or Assignment documents found in %q (scanned %d YAML file(s)) — check --dir",
			dir, len(loaded.Skipped),
		)
	}

	// Step 2 — validate policies
	var validatedPolicies []models.PathPolicy
	invalid := 0
	for _, pp := range pathPolicies {
		if err := engine.ValidatePolicy(pp.Policy, pp.Path); err != nil {
			fmt.Fprintln(os.Stderr, colorRed(formatFileError("Policy", pp.Policy.Metadata.Name, pp.Path, err)))
			invalid++
		} else {
			fmt.Println(colorGreen(fmt.Sprintf("✔ Policy %q (%s)", pp.Policy.Metadata.Name, pp.Path)))
			validatedPolicies = append(validatedPolicies, pp)
		}
	}

	// Step 3 — validate assignments
	var validatedAssignments []models.PathAssignment
	for _, pa := range pathAssignments {
		if err := engine.ValidateAssignment(pa.Assignment, pa.Path); err != nil {
			fmt.Fprintln(os.Stderr, colorRed(formatFileError("Assignment", pa.Assignment.Metadata.Name, pa.Path, err)))
			invalid++
		} else {
			fmt.Println(colorGreen(fmt.Sprintf("✔ Assignment %q (%s)", pa.Assignment.Metadata.Name, pa.Path)))
			validatedAssignments = append(validatedAssignments, pa)
		}
	}

	if invalid > 0 {
		return nil, fmt.Errorf("Validation failed for %d file(s)", invalid)
	}

	// Step 4 — resolve
	resolved, resolveErrs := engine.Resolve(validatedPolicies, validatedAssignments)
	if len(resolveErrs) > 0 {
		for _, e := range resolveErrs {
			fmt.Fprintln(os.Stderr, colorRed("✖ Resolve: "+e.Error()))
		}
		return nil, fmt.Errorf("Resolve failed with %d error(s)", len(resolveErrs))
	}

	// Backstop: assignments that resolve to nothing must not report success.
	// A silently-skipped assignment would otherwise leave every one of its
	// targets unprotected while validate, diff and apply all exit 0.
	if len(validatedAssignments) > 0 && len(resolved) == 0 {
		return nil, fmt.Errorf(
			"%d assignment(s) loaded but no targets resolved — every assignment was skipped, "+
				"which would leave the targets they declare unprotected",
			len(validatedAssignments),
		)
	}

	return resolved, nil
}
