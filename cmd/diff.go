package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

// Status colors stay on the standard 16-color palette so they follow whatever
// the user has themed their terminal to, and degrade gracefully over SSH.
// Brand accents below use truecolor — kåda and each provider should look the
// same everywhere.
var (
	colorGreen   = color.New(color.FgGreen).SprintFunc()
	colorRed     = color.New(color.FgRed).SprintFunc()
	colorRedBold = color.New(color.FgRed, color.Bold).SprintFunc()
	colorYellow  = color.New(color.FgYellow).SprintFunc()
	colorGray    = color.New(color.FgHiBlack).SprintFunc()

	colorKada     = color.RGB(245, 158, 11).SprintFunc()
	colorKadaBold = color.RGB(245, 158, 11).Add(color.Bold).SprintFunc()

	// Section labels inside a target block are structure, not chrome — bold
	// alone reads as a heading without spending a color on it.
	colorSection = color.New(color.Bold).SprintFunc()

	// Provider accents — used only for resource headers, so a header is
	// findable when scrolling back through a long run.
	colorAzure = color.RGB(80, 230, 255).Add(color.Bold).SprintFunc() // #50E6FF
	colorAWS   = color.RGB(140, 79, 255).Add(color.Bold).SprintFunc() // #8C4FFF, stand-in
)

// providerColor returns the accent for a provider's resource header. Unknown
// providers fall back to kåda's own color rather than going uncolored.
func providerColor(provider string) func(...any) string {
	switch provider {
	case "azure":
		return colorAzure
	case "aws":
		return colorAWS
	default:
		return colorKadaBold
	}
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local backup policies against live configuration",
	Long: `Loads Policy and Assignment YAML files and compares each target's desired
state against the currently configured backup policy. Outputs a structured
change plan without modifying anything.`,
	Args:        cobra.NoArgs,
	RunE:        runDiff,
	Annotations: map[string]string{annotationExitCodes: "true"},
}

func runDiff(cmd *cobra.Command, args []string) error {
	fmt.Println(colorKadaBold("──────────────── kåda — backup policy as code ────────────────"))
	dir := viper.GetString("dir")

	// Load/validate/resolve failures are returned verbatim — they already carry
	// their own count, and folding them into the diff error count would report
	// N failing files as a single error.
	resolved, err := loadAndValidate(dir)
	if err != nil {
		return failure(err)
	}

	drifted, errs := diffTargets(cmd.Context(), resolved)

	// Errors outrank drift. A run that could not reach half its targets knows
	// nothing about whether those targets are compliant, so it must not report
	// the clean-but-drifted verdict a pipeline treats as "diff worked".
	if len(errs) > 0 {
		return failuref("Diff completed with %d error(s)", len(errs))
	}
	if drifted > 0 {
		return driftf("%d of %d target(s) differ from desired state", drifted, len(resolved))
	}
	return nil
}

// diffTargets diffs every resolved target and returns how many of them differ
// from their live state, alongside any errors encountered.
func diffTargets(ctx context.Context, resolved []models.ResolvedTarget) (int, []error) {
	var errs []error

	var azureTargets []models.ResolvedTarget

	// Every target must land in a bucket that is actually diffed. An "aws"
	// bucket used to be collected here and then never diffed, which would have
	// counted those targets as compliant without ever looking at them — add
	// the case here only together with the call that consumes it.
	for _, rt := range resolved {
		switch rt.Provider {
		case "azure":
			azureTargets = append(azureTargets, rt)
		default:
			errs = append(errs, fmt.Errorf("Provider %q is not yet supported", rt.Provider))
			continue
		}
	}

	results, azureErrs := diffAzureTargets(ctx, azureTargets)
	errs = append(errs, azureErrs...)

	// The results were previously discarded here, which left runDiff with no
	// way to tell a clean environment from a drifted one.
	drifted := 0
	for _, r := range results {
		if r.HasChanges() {
			drifted++
		}
	}

	return drifted, errs
}

func diffAzureTargets(ctx context.Context, targets []models.ResolvedTarget) ([]azure_engine.AzureDiffResult, []error) {
	var errs []error
	var results []azure_engine.AzureDiffResult

	fmt.Println(colorKada(fmt.Sprintf("Resolved %d target(s) — connecting to Azure", len(targets))))

	credAny, err := engine.Authenticate("azure")
	if err != nil {
		return nil, []error{fmt.Errorf("Authentication failed: %w", err)}
	}

	// Casting anyCred to the Azure specific credential type
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
		fmt.Println(providerColor(rt.Provider)(fmt.Sprintf("\n──── %s/%s (%s) ────", resourceGroup, resourceName, resourceType)))

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
	fmt.Println(colorSection("POLICY DEFINITION:"))
	if r.RetentionMissing {
		fmt.Println(colorYellow("  (does not exist — will be created)"))
	} else if len(r.RetentionApply) > 0 {
		for key, desired := range r.RetentionApply {
			printDiffLine(key, r.RetentionOverwrite[key], desired)
		}
	} else {
		fmt.Println(colorGreen("  NO CHANGES"))
	}

	fmt.Println(colorSection("PROTECTED ITEM:"))
	if len(r.ToApply) > 0 {
		for key, desired := range r.ToApply {
			printDiffLine(key, r.Overwrite[key], desired)
		}
	} else {
		fmt.Println(colorGreen("  NO CHANGES"))
	}
}

// printDiffLine renders one "key: current => desired" row.
func printDiffLine(key string, current, desired any) {
	fmt.Printf("  %s:\t%s => %s\n",
		key,
		formatOldVal(shortenARMID(current)),
		colorGreen(fmt.Sprintf("%v", shortenARMID(desired))))
}

// shortenARMID trims an Azure resource ID to its final segment for display.
// Two policy IDs that differ only in the last segment are unreadable side by
// side. This is presentation only — the diff maps always compare full IDs, so
// two same-named policies in different vaults still register as a change.
func shortenARMID(v any) any {
	s, ok := v.(string)
	if !ok || !strings.HasPrefix(s, "/subscriptions/") {
		return v
	}
	if i := strings.LastIndex(s, "/"); i >= 0 && i < len(s)-1 {
		return s[i+1:]
	}
	return v
}

// formatOldVal renders the value being replaced. Yellow means a real value is
// about to be overwritten; gray means there is nothing there yet, which is
// most of a first-run diff and shouldn't read as a wall of warnings.
func formatOldVal(v any) string {
	s := formatVal(v)
	if s == "(not set)" {
		return colorGray(s)
	}
	return colorYellow(s)
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
