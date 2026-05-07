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

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local backup policies against live Azure configuration",
	Long: `Loads Policy and Assignment YAML files and compares each target's desired
state against the currently configured Azure backup policy. Outputs a structured
change plan without modifying anything.`,
	RunE: runDiff,
}

var green = color.New(color.FgGreen).SprintFunc()
var cyan = color.New(color.FgHiCyan).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()

func runDiff(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("dir")
	_, _, errs := diffTargets(dir, false)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", e)
		}
		return fmt.Errorf("diff completed with %d error(s)", len(errs))
	}
	return nil
}

// diffTargets loads, validates, resolves, and diffs all targets.
// When applying=true the per-target diff output is suppressed.
// Returns (equalMaps, diffMaps, errors).
func diffTargets(dir string, applying bool) (
	[]map[string]interface{},
	[]map[string]interface{},
	[]error,
) {
	var errs []error
	var equalMaps []map[string]interface{}
	var diffMaps []map[string]interface{}

	resolved, _, err := loadAndValidate(dir, true)
	if err != nil {
		return nil, nil, []error{err}
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		subscriptionID = viper.GetString("subscription")
	}
	if subscriptionID == "" {
		return nil, nil, []error{fmt.Errorf("subscription ID not set (use --subscription or AZURE_SUBSCRIPTION_ID)")}
	}

	cred, err := engine.Authenticate()
	if err != nil {
		return nil, nil, []error{err}
	}

	for _, rt := range resolved {
		subID := rt.SubscriptionID
		if subID == "" {
			subID = subscriptionID
		}

		var equalSettings, diff, overwrite map[string]interface{}

		switch rt.Target.Type {
		case "azure_vm":
			remote, err := engine.GetBackupPolicyVM(
				cred, subID,
				rt.Target.ResourceGroup,
				rt.Policy.Spec.Vault.Name,
				rt.Target.ResourceName,
			)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: %w", rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}
			equalSettings, diff, _, overwrite, err = engine.DiffPolicy(rt, remote)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: diff failed: %w", rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}

		case "azure_sql", "azure_sql_vm":
			remote, err := engine.GetBackupPolicyDB(
				cred, subID,
				rt.Target.ResourceGroup,
				rt.Policy.Spec.Vault.Name,
				rt.Policy.Metadata.Name,
			)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: %w", rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}
			equalSettings, diff, _, overwrite, err = engine.DiffDBPolicy(rt, remote)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: diff failed: %w", rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}

		case "azure_disk":
			errs = append(errs, fmt.Errorf("%s/%s: azure_disk diff not yet implemented",
				rt.Target.ResourceGroup, rt.Target.ResourceName))
			continue

		default:
			errs = append(errs, fmt.Errorf("%s/%s: unsupported target type %q",
				rt.Target.ResourceGroup, rt.Target.ResourceName, rt.Target.Type))
			continue
		}

		if !applying {
			printDiff(rt, equalSettings, diff, overwrite)
		}

		equalMaps = append(equalMaps, equalSettings)
		diffMaps = append(diffMaps, diff)
	}

	return equalMaps, diffMaps, errs
}

func printDiff(rt models.ResolvedTarget, equal, diff, overwrite map[string]interface{}) {
	fmt.Printf("\n──── %s/%s (%s) ────\n",
		rt.Target.ResourceGroup, rt.Target.ResourceName, rt.Target.Type)

	if len(diff) > 0 {
		fmt.Println("CHANGES:")
		for key, desired := range diff {
			fmt.Printf("  %s:\t%v => %v\n", key, formatVal(overwrite[key]), desired)
		}
	} else {
		fmt.Println(green("  NO CHANGES"))
	}

	if len(equal) > 0 {
		fmt.Println("UNCHANGED:")
		for key, val := range equal {
			fmt.Printf("  %s: %v\n", key, val)
		}
	}
}

func formatVal(v interface{}) string {
	if v == nil {
		return "(not set)"
	}
	if s, ok := v.(string); ok && s == "" {
		return "(not set)"
	}
	return fmt.Sprintf("%v", v)
}
