package azure_engine

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	azure_external "github.com/dbjorklund/backup-policy-as-code/external/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

// DiffResult holds the outcome of a single target diff.
type DiffResult struct {
	Target             any
	Equal              map[string]any
	ToApply            map[string]any
	Overwrite          map[string]any
	RetentionEqual     map[string]any
	RetentionApply     map[string]any
	RetentionOverwrite map[string]any
	RetentionMissing   bool
}

// ResourceMeta returns common fields for display/logging.
func (d DiffResult) ResourceMeta() (resourceGroup, resourceName, resourceType string) {
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

func DiffVM(ctx context.Context, cred *azidentity.DefaultAzureCredential, vm azure.ResolvedVM) (DiffResult, error) {
	result := DiffResult{Target: vm}

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: vm.SubscriptionID,
	}

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
		retEqual, retToApply, _, retOverwrite, err := DiffVMRetention(vm, remoteLTR)
		if err != nil {
			return result, fmt.Errorf("%s/%s: retention diff failed: %w",
				vm.ResourceGroup, vm.ResourceName, err)
		}
		result.RetentionEqual = retEqual
		result.RetentionApply = retToApply
		result.RetentionOverwrite = retOverwrite
	}

	// --- Protected item (VM assignment to the policy) ---
	remote, err := client.GetBackupPolicyVM(ctx, vm.ResourceGroup, vm.VaultName, vm.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get protected item state: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}

	equal, toApply, _, toOverwrite, err := DiffVMPolicy(vm, *remote)
	if err != nil {
		return result, fmt.Errorf("%s/%s: diff failed: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}

	result.Equal = equal
	result.ToApply = toApply
	result.Overwrite = toOverwrite
	return result, nil
}

func DiffDisk(ctx context.Context, cred *azidentity.DefaultAzureCredential, disk azure.ResolvedDisk) (DiffResult, error) {
	result := DiffResult{Target: disk}

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: disk.SubscriptionID,
	}

	existingDisk, err := client.GetDisk(ctx, disk.ResourceGroup, disk.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get disk: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if existingDisk == nil {
		return result, fmt.Errorf("%s/%s: disk does not exist",
			disk.ResourceGroup, disk.ResourceName)
	}

	remotePolicy, err := client.GetBackupPolicyDefinitionDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get policy definition: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if remotePolicy == nil {
		result.RetentionMissing = true
	} else {
		retEqual, retToApply, _, retOverwrite, err := DiffDiskRetention(disk, remotePolicy)
		if err != nil {
			return result, fmt.Errorf("%s/%s: disk retention diff failed: %w",
				disk.ResourceGroup, disk.ResourceName, err)
		}
		result.RetentionEqual = retEqual
		result.RetentionApply = retToApply
		result.RetentionOverwrite = retOverwrite
	}

	remote, err := client.GetBackupPolicyDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get backup instance state: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}
	if remote == nil {
		result.Equal = map[string]any{}
		result.ToApply = map[string]any{"policyID": disk.PolicyName}
		result.Overwrite = map[string]any{}
		return result, nil
	}

	equal, toApply, _, toOverwrite, err := DiffDiskPolicy(disk, *remote)
	if err != nil {
		return result, fmt.Errorf("%s/%s: diff failed: %w",
			disk.ResourceGroup, disk.ResourceName, err)
	}

	result.Equal = equal
	result.ToApply = toApply
	result.Overwrite = toOverwrite
	return result, nil
}

func DiffSqlDb(ctx context.Context, cred *azidentity.DefaultAzureCredential, db azure.ResolvedSqlDb) (DiffResult, error) {
	result := DiffResult{Target: db}

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: db.SubscriptionID,
	}

	existingDb, err := client.GetSqlDb(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get SQL database: %w",
			db.ResourceGroup, db.ResourceName, err)
	}
	if existingDb == nil {
		return result, fmt.Errorf("%s/%s: SQL database does not exist",
			db.ResourceGroup, db.ResourceName)
	}

	remoteSTR, err := client.GetSqlDbSTR(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get STR policy: %w",
			db.ResourceGroup, db.ResourceName, err)
	}

	remoteLTR, err := client.GetSqlDbLTR(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return result, fmt.Errorf("%s/%s: failed to get LTR policy: %w",
			db.ResourceGroup, db.ResourceName, err)
	}

	if remoteSTR == nil {
		result.RetentionMissing = true
	} else {
		retEqual, retToApply, _, retOverWrite, err := DiffSqlDbRetention(db, remoteSTR, remoteLTR, existingDb)
		if err != nil {
			return result, fmt.Errorf("%s/%s: retention diff failed: %w",
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
