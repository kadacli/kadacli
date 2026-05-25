package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Policy and Assignment YAML files without connecting to Azure",
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("dir")
	_, _, err := loadAndValidate(dir, false)
	return err
}

// loadAndValidate loads, parses, and resolves all YAML files in dir.
// Returns the resolved targets and any fatal error. When applying=true the
// "All policies checked" summary line is suppressed.
func loadAndValidate(dir string, applying bool) ([]models.ResolvedTarget, []models.Policy, error) {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgHiCyan).SprintFunc()

	policies, assignments, loadErrs := engine.LoadAll(dir)
	for _, e := range loadErrs {
		fmt.Fprintln(os.Stderr, red("✖ "+e.Error()))
	}

	resolved, resolveErrs := engine.Resolve(policies, assignments)
	if len(resolveErrs) > 0 {
		for _, e := range resolveErrs {
			fmt.Fprintln(os.Stderr, red("✖ "+e.Error()))
		}
		return nil, policies, fmt.Errorf("validation failed with %d error(s)", len(resolveErrs))
	}

	for _, p := range policies {
		fmt.Println(green(fmt.Sprintf("✔ Policy %q is valid", p.Metadata.Name)))
	}
	for _, a := range assignments {
		fmt.Println(green(fmt.Sprintf("✔ Assignment %q is valid", a.Metadata.Name)))
	}

	if !applying {
		fmt.Println(cyan(fmt.Sprintf("All files checked — %d policy/assignment file(s) OK", len(policies)+len(assignments))))
	}

	return resolved, policies, nil
}
