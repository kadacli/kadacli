package azure_engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	azure_external "github.com/dbjorklund/backup-policy-as-code/external/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

// policyResultMessage reports what actually happened to a policy definition.
// Apply reports results, not intent — the create/update distinction matters
// when you need to know whether an existing policy was overwritten.
func policyResultMessage(isNew bool, policyName string) string {
	if isNew {
		return fmt.Sprintf("✔ Created policy %q", policyName)
	}
	return fmt.Sprintf("✔ Updated policy %q", policyName)
}

func ApplyVM(ctx context.Context, cred *azidentity.DefaultAzureCredential, vm azure.ResolvedVM) (int, []string, error) {
	label := vm.ResourceGroup + "/" + vm.ResourceName
	applied := 0
	var messages []string

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: vm.SubscriptionID,
	}

	existingVM, err := client.GetVM(ctx, vm.ResourceGroup, vm.ResourceName)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: Failed to get VM: %w", label, err)
	}
	if existingVM == nil {
		return 0, nil, fmt.Errorf("%s: VM does not exist", label)
	}
	if err := client.CheckVMVaultRegion(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, existingVM); err != nil {
		return 0, nil, fmt.Errorf("%s: %w", label, err)
	}

	remotePolicy, err := client.GetBackupPolicyDefinitionVM(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, vm.PolicyName)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: Failed to get policy definition: %w", label, err)
	}

	policyNeedsApply := false
	policyIsNew := remotePolicy == nil
	if policyIsNew {
		policyNeedsApply = true
	} else {
		remoteLTR, ok := remotePolicy.RetentionPolicy.(*armrecoveryservicesbackup.LongTermRetentionPolicy)
		if !ok {
			return 0, nil, fmt.Errorf("%s: Remote retention policy is not LongTermRetentionPolicy", label)
		}

		_, schedDiff, _, _, err := DiffVMSchedule(vm, remotePolicy.SchedulePolicy)
		if err != nil {
			return 0, nil, fmt.Errorf("%s: Schedule diff failed: %w", label, err)
		}
		_, retDiff, _, _, err := DiffVMRetention(vm, remoteLTR)
		if err != nil {
			return 0, nil, fmt.Errorf("%s: Policy definition diff failed: %w", label, err)
		}
		_, typeDiff, typeOverwrite := DiffVMPolicyType(vm, remotePolicy.PolicyType)
		if len(typeDiff) > 0 {
			return 0, nil, fmt.Errorf(
				"%s: Cannot change policy %q from %v to %v — Azure does not support updating an existing policy's type in place; create a new policy under a different metadata.name instead",
				label, vm.PolicyName, typeOverwrite["policyType"], typeDiff["policyType"],
			)
		}

		if len(retDiff) > 0 || len(schedDiff) > 0 {
			policyNeedsApply = true
		}

	}

	if policyNeedsApply {
		if err := client.CreateOrUpdateBackupPolicyVM(ctx, vm); err != nil {
			return 0, messages, fmt.Errorf("%s: Failed to upsert policy definition: %w", label, err)
		}
		messages = append(messages, policyResultMessage(policyIsNew, vm.PolicyName))
		applied++
	}

	remote, err := client.GetBackupPolicyVM(ctx, vm.ResourceGroup, vm.VaultName, vm.ResourceName)
	if err != nil {
		return applied, messages, fmt.Errorf("%s: Failed to get protected item state: %w", label, err)
	}

	if azure_external.IsVMBackupPaused(remote) {
		return applied, nil, fmt.Errorf(
			"%s: Backup is currently paused/stopped — resume it in the Azure Portal (or via az backup protection resume) before this tool can manage its policy assignment",
			label,
		)
	}

	if remote == nil {
		if err := client.ApplyVM(ctx, vm); err != nil {
			return applied, messages, fmt.Errorf("%s: Apply failed: %w", label, err)
		}
		messages = append(messages, fmt.Sprintf("✔ Applied policy %q to %s", vm.PolicyName, label))
		return applied + 1, messages, nil
	}

	_, itemDiff, _, _, err := DiffVMPolicy(vm, *remote)
	if err != nil {
		return applied, messages, fmt.Errorf("%s: Protected item diff failed: %w", label, err)
	}

	if vm.Spec.PolicyType == "Standard" {
		policyType, err := client.CurrentVMPolicyType(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, remote)
		if err != nil {
			return applied, messages, fmt.Errorf("%s: Failed to check current policy type: %w", label, err)
		}
		if policyType == armrecoveryservicesbackup.IAASVMPolicyTypeV2 {
			return applied, messages, fmt.Errorf(
				"%s: Cannot assign a Standard policy — this VM is currently protected by an Enhanced (V2) policy, and Azure does not support downgrading VM protection from Enhanced to Standard",
				label,
			)
		}
	}

	if len(itemDiff) == 0 {
		messages = append(messages, fmt.Sprintf("✔ %s — Backup instance already assigned correctly", label))
		return applied, messages, nil
	}

	if err := client.ApplyVM(ctx, vm); err != nil {
		return applied, messages, fmt.Errorf("%s: Apply failed: %w", label, err)
	}

	messages = append(messages, fmt.Sprintf("✔ Applied policy %q to %s", vm.PolicyName, label))
	return applied + 1, messages, nil
}

func ApplyDisk(ctx context.Context, cred *azidentity.DefaultAzureCredential, disk azure.ResolvedDisk) (int, []string, error) {
	label := disk.ResourceGroup + "/" + disk.ResourceName
	applied := 0
	var messages []string

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: disk.SubscriptionID,
	}

	existingDisk, err := client.GetDisk(ctx, disk.ResourceGroup, disk.ResourceName)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: Failed to get disk: %w", label, err)
	}
	if existingDisk == nil {
		return 0, nil, fmt.Errorf("%s: Disk does not exist", label)
	}

	remotePolicy, err := client.GetBackupPolicyDefinitionDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: Failed to get policy definition: %w", label, err)
	}

	policyNeedsApply := false
	policyIsNew := remotePolicy == nil
	if policyIsNew {
		policyNeedsApply = true
	} else {
		_, retDiff, _, _, err := DiffDiskPolicyDefinition(disk, remotePolicy)
		if err != nil {
			return 0, nil, fmt.Errorf("%s: Policy definition diff failed: %w", label, err)
		}
		if len(retDiff) > 0 {
			policyNeedsApply = true
		}
	}

	if policyNeedsApply {
		if err := client.CreateOrUpdateBackupPolicyDisk(ctx, disk); err != nil {
			return 0, messages, fmt.Errorf("%s: Failed to upsert policy definition: %w", label, err)
		}
		messages = append(messages, policyResultMessage(policyIsNew, disk.PolicyName))
		applied++
	}

	remote, err := client.GetBackupPolicyDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.ResourceName)
	if err != nil {
		return applied, messages, fmt.Errorf("%s: Failed to get backup instance state: %w", label, err)
	}

	if remote == nil {
		if err := client.ApplyDisk(ctx, disk); err != nil {
			return applied, messages, fmt.Errorf("%s: Apply failed: %w", label, err)
		}
		messages = append(messages, fmt.Sprintf("✔ Applied policy %q to %s", disk.PolicyName, label))
		return applied + 1, messages, nil
	}

	_, itemDiff, _, _, err := DiffDiskPolicy(disk, *remote)
	if err != nil {
		return applied, messages, fmt.Errorf("%s: Backup instance diff failed: %w", label, err)
	}

	if len(itemDiff) == 0 {
		messages = append(messages, fmt.Sprintf("✔ %s — Protected item already assigned correctly", label))
		return applied, messages, nil
	}

	if err := client.ApplyDisk(ctx, disk); err != nil {
		return applied, messages, fmt.Errorf("%s: Apply failed: %w", label, err)
	}

	messages = append(messages, fmt.Sprintf("✔ Applied policy %q to %s", disk.PolicyName, label))
	return applied + 1, messages, nil
}

func ApplySqlDb(ctx context.Context, cred *azidentity.DefaultAzureCredential, db azure.ResolvedSqlDb) (int, []string, error) {
	label := db.ResourceGroup + "/" + db.ResourceName
	applied := 0
	var messages []string

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: db.SubscriptionID,
	}

	existingDb, err := client.GetSqlDb(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: Failed to get SQL database: %w", label, err)
	}
	if existingDb == nil {
		return 0, nil, fmt.Errorf("%s: SQL database does not exist", label)
	}

	remoteSTR, err := client.GetSqlDbSTR(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: Failed to get STR policy: %w", label, err)
	}

	remoteLTR, err := client.GetSqlDbLTR(ctx, db.ResourceGroup, db.ServerName, db.ResourceName)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: Failed to get LTR policy: %w", label, err)
	}

	_, retDiff, _, _, err := DiffSqlDbRetention(db, remoteSTR, remoteLTR, existingDb)
	if err != nil {
		return 0, nil, fmt.Errorf("%s: Policy definition diff failed: %w", label, err)
	}

	// Check if any STR keys changed
	strChanged := false
	for k := range retDiff {
		if strings.HasPrefix(k, "str.") {
			strChanged = true
			break
		}
	}

	// Check if any LTR keys changed
	ltrChanged := false
	for k := range retDiff {
		if strings.HasPrefix(k, "ltr.") {
			ltrChanged = true
			break
		}
	}
	redundancyChanged := false
	for k := range retDiff {
		if strings.HasPrefix(k, "db.") {
			redundancyChanged = true
			break
		}
	}

	if len(retDiff) == 0 {
		messages = append(messages, fmt.Sprintf("✔ %s — Already in desired state", label))
		return applied, messages, nil
	}

	if strChanged {
		if err := client.ApplySqlDbSTR(ctx, db); err != nil {
			return 0, messages, fmt.Errorf("%s: Failed to apply STR policy: %w", label, err)
		}
		messages = append(messages, fmt.Sprintf("✔ Applied STR policy to %q", db.ResourceName))
		applied++
	}
	if redundancyChanged {
		if err := client.ApplySqlDbRedundancy(ctx, db); err != nil {
			return applied, messages, fmt.Errorf("%s: Failed to update redundancy: %w", label, err)
		}
		messages = append(messages, fmt.Sprintf("✔ Updated redundancy to %q for %q", db.Spec.Redundancy, db.ResourceName))
		applied++
	}
	if ltrChanged {
		ltr := db.Spec.LongTermRetention
		// A diff on any ltr.* key means the remote LTR policy must be
		// reconciled — whether that's setting/updating retention or
		// clearing it. When all local retentions are 0 (no
		// long_term_retention block), ApplySqlDbLTR sends P0W/P0M/P0Y,
		// which is how Azure removes an existing LTR policy. Skipping the
		// call in that case would silently leave stale LTR on the DB.
		clearing := ltr.WeeklyRetention == 0 && ltr.MonthlyRetention == 0 && ltr.YearlyRetention == 0
		if err := client.ApplySqlDbLTR(ctx, db); err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.ErrorCode == "LtrConfigPolicyUnsupportedIfAutoPauseEnabled" {
				messages = append(messages, fmt.Sprintf("⚠ LTR not supported for %q (serverless with auto-pause) — skipping", db.ResourceName))
			} else {
				return applied, messages, fmt.Errorf("%s: Failed to apply LTR policy: %w", label, err)
			}
		} else {
			if clearing {
				messages = append(messages, fmt.Sprintf("✔ Cleared LTR policy for %q", db.ResourceName))
			} else {
				messages = append(messages, fmt.Sprintf("✔ Applied LTR policy to %q", db.ResourceName))
			}
			applied++
		}
	}

	return applied, messages, nil
}
