package azure_engine

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	azure_external "github.com/kadacli/kadacli/external/azure"
	"github.com/kadacli/kadacli/internal/models"
	"github.com/kadacli/kadacli/internal/models/azure"
)

// DiffResult holds the outcome of a single target diff for Azure
type AzureDiffResult struct {
	models.DiffResult
}

// sortedKeys returns a map's keys in a stable order, so an error message
// listing changed fields reads the same on every run.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Adding a method to models.DiffResult
// ResourceMeta returns common fields for display/logging.
func (d AzureDiffResult) ResourceMeta() (resourceGroup, resourceName, resourceType string) {
	switch t := d.Target.(type) {
	case azure.ResolvedVM:
		return t.ResourceGroup, t.ResourceName, t.Resource.Type
	case azure.ResolvedDisk:
		return t.ResourceGroup, t.ResourceName, t.Resource.Type
	case azure.ResolvedSqlDb:
		return t.ResourceGroup, t.ResourceName, t.Resource.Type
	default:
		return "unknown", "unknown", "unknown"
	}
}

func DiffVM(ctx context.Context, cred *azidentity.DefaultAzureCredential, vm azure.ResolvedVM) (AzureDiffResult, error) {
	result := AzureDiffResult{DiffResult: models.DiffResult{Target: vm}}

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: vm.SubscriptionID,
	}

	existingVM, err := client.GetVM(ctx, vm.ResourceGroup, vm.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get VM: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}
	if existingVM == nil {
		return result, fmt.Errorf("%s/%s: VM does not exist",
			vm.ResourceGroup, vm.ResourceName)
	}
	if err := client.CheckVMVaultRegion(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, existingVM); err != nil {
		return result, fmt.Errorf("%s/%s: %w", vm.ResourceGroup, vm.ResourceName, err)
	}

	remotePolicy, err := client.GetBackupPolicyDefinitionVM(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, vm.PolicyName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get policy definition: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}
	if remotePolicy == nil {
		result.RetentionMissing = true
	} else {
		remoteLTR, ok := remotePolicy.RetentionPolicy.(*armrecoveryservicesbackup.LongTermRetentionPolicy)
		if !ok {
			return result, fmt.Errorf("%s/%s: Remote retention policy is not LongTermRetentionPolicy",
				vm.ResourceGroup, vm.ResourceName)
		}
		retEqual, retToApply, _, retOverwrite, err := DiffVMRetention(vm, remoteLTR)
		if err != nil {
			return result, fmt.Errorf("%s/%s: Retention diff failed: %w",
				vm.ResourceGroup, vm.ResourceName, err)
		}
		schedEqual, schedToApply, _, schedOverwrite, err := DiffVMSchedule(vm, remotePolicy.SchedulePolicy)
		if err != nil {
			return result, fmt.Errorf("%s/%s: Schedule diff failed: %w",
				vm.ResourceGroup, vm.ResourceName, err)
		}

		typeEqual, typeToApply, typeOverwrite := DiffVMPolicyType(vm, remotePolicy.PolicyType)
		if len(typeToApply) > 0 {
			return result, fmt.Errorf(
				"%s/%s: Cannot change policy %q from %v to %v — Azure does not support updating an existing policy's type in place; create a new policy under a different metadata.name instead",
				vm.ResourceGroup, vm.ResourceName, vm.PolicyName, typeOverwrite["policyType"], typeToApply["policyType"],
			)
		}

		result.RetentionEqual = mergeMaps(retEqual, schedEqual, typeEqual)
		result.RetentionApply = mergeMaps(retToApply, schedToApply, typeToApply)
		result.RetentionOverwrite = mergeMaps(retOverwrite, schedOverwrite, typeOverwrite)
	}

	// --- Protected item (VM assignment to the policy) ---
	remote, err := client.GetBackupPolicyVM(ctx, vm.ResourceGroup, vm.VaultName, vm.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get protected item state: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}
	if azure_external.IsVMBackupPaused(remote) {
		return result, fmt.Errorf(
			"%s/%s: Backup is currently paused/stopped — resume it in the Azure Portal (or via az backup protection resume) before this tool can manage its policy assignment",
			vm.ResourceGroup, vm.ResourceName,
		)
	}
	if remote == nil {
		// Not yet registered as a protected item — everything needs to be applied
		result.Equal = map[string]any{}
		result.ToApply = ResolvedVMToMap(vm)
		result.Overwrite = map[string]any{}
		return result, nil
	}

	equal, toApply, _, toOverwrite, err := DiffVMPolicy(vm, *remote)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Diff failed: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}

	if len(toApply) > 0 && vm.Spec.PolicyType == "Standard" {
		policyType, err := client.CurrentVMPolicyType(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, remote)
		if err != nil {
			return result, fmt.Errorf("%s/%s: Failed to check current policy type: %w",
				vm.ResourceGroup, vm.ResourceName, err)
		}
		if policyType == armrecoveryservicesbackup.IAASVMPolicyTypeV2 {
			return result, fmt.Errorf(
				"%s/%s: Cannot assign a Standard policy — this VM is currently protected by an Enhanced (V2) policy, and Azure does not support downgrading VM protection from Enhanced to Standard",
				vm.ResourceGroup, vm.ResourceName,
			)
		}
	}

	result.Equal = equal
	result.ToApply = toApply
	result.Overwrite = toOverwrite
	return result, nil
}

func DiffDisk(ctx context.Context, cred *azidentity.DefaultAzureCredential, disk azure.ResolvedDisk) (AzureDiffResult, error) {
	result := AzureDiffResult{DiffResult: models.DiffResult{Target: disk}}

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: disk.SubscriptionID,
	}

	existingDisk, err := client.GetDisk(ctx, disk.ResourceGroup, disk.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get disk: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if existingDisk == nil {
		return result, fmt.Errorf("%s/%s: Disk does not exist",
			disk.ResourceGroup, disk.ResourceName)
	}

	remotePolicy, err := client.GetBackupPolicyDefinitionDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get policy definition: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if remotePolicy == nil {
		result.RetentionMissing = true
	} else {
		retEqual, retToApply, _, retOverwrite, err := DiffDiskPolicyDefinition(disk, remotePolicy)
		if err != nil {
			return result, fmt.Errorf("%s/%s: Disk policy definition diff failed: %w",
				disk.ResourceGroup, disk.ResourceName, err)
		}
		result.RetentionEqual = retEqual
		result.RetentionApply = retToApply
		result.RetentionOverwrite = retOverwrite
	}

	remote, err := client.GetBackupPolicyDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get backup instance state: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if remote == nil {
		// No backup instance yet, so the whole local map is a change. Reuse
		// resolvedDiskToMap rather than the bare policy name so "policyID"
		// always holds a full ARM ID, whether or not the disk is protected.
		result.Equal = map[string]any{}
		result.ToApply = resolvedDiskToMap(disk)
		result.Overwrite = map[string]any{}
		return result, nil
	}

	equal, toApply, _, toOverwrite, err := DiffDiskPolicy(disk, *remote)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Diff failed: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}

	// Azure disk backup policies are immutable once a backup instance is
	// protected by them — neither updatable in place nor deletable while in
	// use (see CreateOrUpdateBackupPolicyDisk). If this disk is already on this
	// policy and the definition has changed, apply cannot succeed; say so here
	// rather than after a failed round trip.
	if _, onThisPolicy := equal["policyID"]; onThisPolicy && len(result.RetentionApply) > 0 {
		return result, fmt.Errorf(
			"%s/%s: Policy %q is already protecting this disk and its definition has changed (%s). "+
				"Azure does not support updating a disk backup policy in place — give the policy a new "+
				"metadata.name and re-apply to move the disk onto it",
			disk.ResourceGroup, disk.ResourceName, disk.PolicyName, strings.Join(sortedKeys(result.RetentionApply), ", "),
		)
	}

	result.Equal = equal
	result.ToApply = toApply
	result.Overwrite = toOverwrite
	return result, nil
}

func DiffSqlDb(ctx context.Context, cred *azidentity.DefaultAzureCredential, db azure.ResolvedSqlDb) (AzureDiffResult, error) {
	result := AzureDiffResult{DiffResult: models.DiffResult{Target: db}}

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: db.SubscriptionID,
	}

	existingDb, err := client.GetSqlDb(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get SQL database: %w",
			db.ResourceGroup, db.ResourceName, err)
	}
	if existingDb == nil {
		return result, fmt.Errorf("%s/%s: SQL database does not exist",
			db.ResourceGroup, db.ResourceName)
	}

	remoteSTR, err := client.GetSqlDbSTR(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get STR policy: %w",
			db.ResourceGroup, db.ResourceName, err)
	}

	remoteLTR, err := client.GetSqlDbLTR(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: Failed to get LTR policy: %w",
			db.ResourceGroup, db.ResourceName, err)
	}

	if remoteSTR == nil {
		result.RetentionMissing = true
	} else {
		retEqual, retToApply, _, retOverWrite, err := DiffSqlDbRetention(db, remoteSTR, remoteLTR, existingDb)
		if err != nil {
			return result, fmt.Errorf("%s/%s: Retention diff failed: %w",
				db.ResourceGroup, db.ResourceName, err)
		}
		result.RetentionEqual = retEqual
		result.RetentionApply = retToApply
		result.RetentionOverwrite = retOverWrite
	}

	result.Equal = map[string]any{}
	result.ToApply = map[string]any{}
	result.Overwrite = map[string]any{}
	return result, nil
}
