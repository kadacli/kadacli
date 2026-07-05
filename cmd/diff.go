package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	azure_external "github.com/dbjorklund/backup-policy-as-code/external/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

var (
	colorGreen  = color.New(color.FgGreen).SprintFunc()
	colorRed    = color.New(color.FgRed).SprintFunc()
	colorYellow = color.New(color.FgYellow).SprintFunc()
	colorCyan   = color.New(color.FgHiCyan).SprintFunc()
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare local backup policies against live Azure configuration",
	Long: `Loads Policy and Assignment YAML files and compares each target's desired
state against the currently configured Azure backup policy. Outputs a structured
change plan without modifying anything.`,
	RunE: runDiff,
}

func runDiff(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("dir")
	_, errs := diffTargets(cmd.Context(), dir)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, colorRed("✖ "+e.Error()))
		}
		return fmt.Errorf("diff completed with %d error(s)", len(errs))
	}
	return nil
}

// DiffResult holds the outcome of a single target diff.
type DiffResult struct {
	Target           any // azure.ResolvedVM | azure.ResolvedDisk
	Equal            map[string]any
	ToApply          map[string]any
	Overwrite        map[string]any
	RetentionEqual   map[string]any
	RetentionToApply map[string]any
	RetentionMissing bool // true when the policy definition doesn't exist yet
}

// ResourceMeta returns the common fields needed for display/logging,
// regardless of the underlying resource type.
func (d DiffResult) ResourceMeta() (resourceGroup, resourceName, resourceType string) {
	switch t := d.Target.(type) {
	case azure.ResolvedVM:
		return t.ResourceGroup, t.ResourceName, t.Resource.Type
	case azure.ResolvedDisk:
		return t.ResourceGroup, t.ResourceName, t.Resource.Type
	default:
		return "unknown", "unknown", "unknown"
	}
}

// diffTargets loads, validates, resolves, and diffs all targets against live Azure state.
// Returns per-target results and any errors encountered.
func diffTargets(ctx context.Context, dir string) ([]DiffResult, []error) {
	var errs []error
	var results []DiffResult

	resolved, err := loadAndValidate(dir)
	if err != nil {
		return nil, []error{err}
	}

	fmt.Println(colorCyan(fmt.Sprintf("Resolved %d target(s) — connecting to Azure", len(resolved))))

	// Authenticate once; per-target clients share the same credential.
	// TODO: differentiate auth per supplier
	cred, err := azure_external.Authenticate()
	if err != nil {
		return nil, []error{fmt.Errorf("authentication failed: %w", err)}
	}

	for _, rt := range resolved {
		switch rt.Provider {
		case "azure":
			result, err := diffAzureTarget(ctx, cred, rt)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			printDiff(result)
			results = append(results, result)

		default:
			errs = append(errs, fmt.Errorf("provider %q is not yet supported", rt.Provider))
		}
	}

	return results, errs
}

func diffAzureTarget(ctx context.Context, cred *azidentity.DefaultAzureCredential, rt models.ResolvedTarget) (DiffResult, error) {
	// Note that the payload does not have a resource type
	switch payload := rt.Payload.(type) {
	case azure.ResolvedVM:

		client := &azure_external.Client{
			Credential:     cred,
			SubscriptionID: payload.SubscriptionID,
		}
		return diffAzureVM(ctx, client, payload)
	case azure.ResolvedDisk:

		client := &azure_external.Client{
			Credential:     cred,
			SubscriptionID: payload.SubscriptionID,
		}
		return diffAzureDisk(ctx, client, payload)
	default:
		return DiffResult{}, fmt.Errorf("unexpected payload type %T for azure provider", rt.Payload)
	}
}

func diffAzureDisk(ctx context.Context, client *azure_external.Client, disk azure.ResolvedDisk) (DiffResult, error) {
	result := DiffResult{Target: disk}

	// --- 1. Check disk exists ---
	existingDisk, err := client.GetDisk(ctx, disk.ResourceGroup, disk.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get disk: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if existingDisk == nil {
		return result, fmt.Errorf("%s/%s: disk does not exist",
			disk.ResourceGroup, disk.ResourceName)
	}

	// --- 2. Policy definition (retention / schedule) ---
	remotePolicy, err := client.GetBackupPolicyDefinitionDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get policy definition: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if remotePolicy == nil {
		result.RetentionMissing = true
	} else {
		retEqual, retToApply, _, _, err := azure_engine.DiffDiskRetention(disk, remotePolicy)
		if err != nil {
			return result, fmt.Errorf("%s/%s: disk retention diff failed: %w",
				disk.ResourceGroup, disk.ResourceName, err)
		}
		result.RetentionEqual = retEqual
		result.RetentionToApply = retToApply
	}

	// --- 3. Backup instance (disk assignment to the policy) ---
	remote, err := client.GetBackupPolicyDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get backup instance state: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if remote == nil {
		// Not yet configured — everything needs to be applied
		result.Equal = map[string]any{}
		result.ToApply = map[string]any{"policyID": disk.PolicyName}
		result.Overwrite = map[string]any{}
		return result, nil
	}

	equal, toApply, _, toOverwrite, err := azure_engine.DiffDiskPolicy(disk, *remote)
	if err != nil {
		return result, fmt.Errorf("%s/%s: diff failed: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}

	result.Equal = equal
	result.ToApply = toApply
	result.Overwrite = toOverwrite
	return result, nil
}

func diffAzureVM(ctx context.Context, client *azure_external.Client, vm azure.ResolvedVM) (DiffResult, error) {
	result := DiffResult{Target: vm}

	// --- Policy definition (retention / schedule) ---
	remotePolicy, err := client.GetBackupPolicyDefinitionVM(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, vm.PolicyName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get policy definition: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}
	if remotePolicy == nil {
		result.RetentionMissing = true
	} else {
		remoteLTR, ok := remotePolicy.RetentionPolicy.(*armrecoveryservicesbackup.LongTermRetentionPolicy)
		if !ok {
			return result, fmt.Errorf("%s/%s: remote retention policy is not LongTermRetentionPolicy",
				vm.ResourceGroup, vm.ResourceName)
		}
		retEqual, retToApply, _, _, err := azure_engine.DiffVMRetention(vm, remoteLTR)
		if err != nil {
			return result, fmt.Errorf("%s/%s: retention diff failed: %w",
				vm.ResourceGroup, vm.ResourceName, err)
		}
		result.RetentionEqual = retEqual
		result.RetentionToApply = retToApply
	}

	// --- Protected item (VM assignment to the policy) ---
	remote, err := client.GetBackupPolicyVM(ctx, vm.ResourceGroup, vm.VaultName, vm.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get protected item state: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}

	equal, toApply, _, toOverwrite, err := azure_engine.DiffVMPolicy(vm, remote)
	if err != nil {
		return result, fmt.Errorf("%s/%s: diff failed: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}

	result.Equal = equal
	result.ToApply = toApply
	result.Overwrite = toOverwrite
	return result, nil
}

func printDiff(r DiffResult) {
	resourceGroup, resourceName, resourceType := r.ResourceMeta()
	fmt.Println(colorCyan(fmt.Sprintf("\n──── %s/%s (%s) ────",
		resourceGroup, resourceName, resourceType)))

	// Policy definition section
	fmt.Println("POLICY DEFINITION:")
	if r.RetentionMissing {
		fmt.Println(colorYellow("  (does not exist — will be created)"))
	} else if len(r.RetentionToApply) > 0 {
		for key, desired := range r.RetentionToApply {
			fmt.Printf("  %s:\t%s => %s\n",
				key, colorYellow(formatVal(nil)), colorGreen(fmt.Sprintf("%v", desired)))
		}
	} else {
		fmt.Println(colorGreen("  NO CHANGES"))
	}

	// Protected item section
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
