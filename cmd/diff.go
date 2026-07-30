package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

var (
	colorGreen  = color.New(color.FgGreen).SprintFunc()
	colorRed    = color.New(color.FgRed).SprintFunc()
	colorYellow = color.New(color.FgYellow).SprintFunc()
	colorCyan   = color.New(color.FgHiCyan).SprintFunc()
	colorKada   = color.RGB(245, 158, 11).SprintFunc()
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local backup policies against live configuration",
	Long: `Loads Policy and Assignment YAML files and compares each target's desired
state against the currently configured backup policy. Outputs a structured
change plan without modifying anything.`,
	RunE: runDiff,
}

func runDiff(cmd *cobra.Command, args []string) error {
	fmt.Println(colorKada("──────────────── kåda — backup policy as code ────────────────"))
	dir := viper.GetString("dir")
	errs := diffTargets(cmd.Context(), dir)
	if len(errs) > 0 {
		return fmt.Errorf("Diff completed with %d error(s)", len(errs))
	}
	return nil
}

func diffTargets(ctx context.Context, dir string) []error {
	var errs []error

	var azureTargets []models.ResolvedTarget
	var awsTargets []models.ResolvedTarget

	resolved, err := loadAndValidate(dir)
	if err != nil {
		return []error{err}
	}

	for _, rt := range resolved {
		switch rt.Provider {
		case "azure":
			azureTargets = append(azureTargets, rt)
		case "aws":
			awsTargets = append(awsTargets, rt)
		default:
			errs = append(errs, fmt.Errorf("Provider %q is not yet supported", rt.Provider))
			continue
		}
	}

	_, azureErrs := diffAzureTargets(ctx, azureTargets)

	for _, err := range azureErrs {
		errs = append(errs, err)
	}

	return errs

}

func diffAzureTargets(ctx context.Context, targets []models.ResolvedTarget) ([]azure_engine.AzureDiffResult, []error) {
	var errs []error
	var results []azure_engine.AzureDiffResult

	fmt.Println(colorKada(fmt.Sprintf("Resolved %d target(s) — connecting to Azure", len(targets))))

	credAny, err := engine.Authenticate("azure")
	if err != nil {
		return nil, []error{fmt.Errorf("Authentication failed: %w", err)}
	}

	cred, ok := credAny.(*azidentity.DefaultAzureCredential)
	if !ok {
		return nil, []error{fmt.Errorf("Authentication failed: %w", err)}
	}

	for _, rt := range targets {
		var result azure_engine.AzureDiffResult
		var err error

		switch payload := rt.Payload.(type) {
		case azure.ResolvedVM:
			result, err = azure_engine.DiffVM(ctx, cred, payload)
		case azure.ResolvedDisk:
			result, err = azure_engine.DiffDisk(ctx, cred, payload)
		case azure.ResolvedSqlDb:
			result, err = azure_engine.DiffSqlDb(ctx, cred, payload)
		default:
			errs = append(errs, fmt.Errorf("Provider %q is not yet supported", rt.Provider))
			continue
		}

		resourceGroup, resourceName, resourceType := result.ResourceMeta()
		fmt.Println(colorKada(fmt.Sprintf("\n──── %s/%s (%s) ────", resourceGroup, resourceName, resourceType)))

		if err != nil {
			errs = append(errs, err)
			fmt.Fprintln(os.Stderr, colorRed("✖ "+err.Error()))
			continue
		}
		printDiff(result.DiffResult)
		results = append(results, result)
	}

	return results, errs
}

func printDiff(r models.DiffResult) {
	fmt.Println("POLICY DEFINITION:")
	if r.RetentionMissing {
		fmt.Println(colorYellow("  (does not exist — will be created)"))
	} else if len(r.RetentionApply) > 0 {
		for key, desired := range r.RetentionApply {
			fmt.Printf("  %s:\t%s => %s\n",
				key, colorYellow(formatVal(r.RetentionOverwrite[key])), colorGreen(fmt.Sprintf("%v", desired)))
		}
	} else {
		fmt.Println(colorGreen("  NO CHANGES"))
	}

	fmt.Println("PROTECTED ITEM:")
	if len(r.ToApply) > 0 {
		for key, desired := range r.ToApply {
			fmt.Printf("  %s:\t%s => %s\n",
				key, colorYellow(formatVal(r.Overwrite[key])), colorGreen(fmt.Sprintf("%v", desired)))
		}
	} else {
		fmt.Println(colorGreen("  NO CHANGES"))
	}
}

func formatVal(v any) string {
	if v == nil {
		return "(not set)"
	}
	if s, ok := v.(string); ok && s == "" {
		return "(not set)"
	}
	return fmt.Sprintf("%v", v)
}
