package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	azure_external "github.com/dbjorklund/backup-policy-as-code/external/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply backup policies idempotently to Azure",
	Long: `Validates policies, diffs desired state against live Azure, and applies
only the resources that are out of desired state. Safe to run repeatedly.`,
	RunE: runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("dir")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	resolved, err := loadAndValidate(dir)
	if err != nil {
		return err
	}

	// Authenticate once; a per-target Client is constructed using the same credential.
	// TODO: differentiate based on vendor
	cred, err := azure_external.Authenticate()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	var applyErrors []error
	applied := 0

	for _, rt := range resolved {
		switch rt.Provider {
		case "azure":
			n, err := applyAzureTarget(cmd.Context(), cred, rt, dryRun)
			applied += n
			if err != nil {
				applyErrors = append(applyErrors, err)
				fmt.Fprintln(os.Stderr, colorRed("✖ "+err.Error()))
			}
		default:
			applyErrors = append(applyErrors, fmt.Errorf("provider %q is not yet supported", rt.Provider))
		}
	}

	fmt.Printf("\nApplied %d change(s) with %d error(s)\n", applied, len(applyErrors))
	if len(applyErrors) > 0 {
		return fmt.Errorf("apply completed with %d error(s)", len(applyErrors))
	}
	return nil
}

func applyAzureTarget(ctx context.Context, cred *azidentity.DefaultAzureCredential, rt models.ResolvedTarget, dryRun bool) (int, error) {
	switch payload := rt.Payload.(type) {
	case azure.ResolvedVM:
		client := &azure_external.Client{
			Credential:     cred,
			SubscriptionID: payload.SubscriptionID,
		}
		return applyAzureVM(ctx, client, payload, dryRun)
	case azure.ResolvedDisk:
		client := &azure_external.Client{
			Credential:     cred,
			SubscriptionID: payload.SubscriptionID,
		}
		return applyAzureDisk(ctx, client, payload, dryRun)
	default:
		return 0, fmt.Errorf("unexpected payload type %T for azure provider", rt.Payload)
	}
}

func applyAzureVM(ctx context.Context, client *azure_external.Client, vm azure.ResolvedVM, dryRun bool) (int, error) {
	label := vm.ResourceGroup + "/" + vm.ResourceName
	applied := 0

	// --- 1. Policy definition (retention / schedule in the vault) ---
	remotePolicy, err := client.GetBackupPolicyDefinitionVM(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, vm.PolicyName)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get policy definition: %w", label, err)
	}

	policyNeedsApply := false
	if remotePolicy == nil {
		fmt.Printf("  policy %q does not exist in vault — will create\n", vm.PolicyName)
		policyNeedsApply = true
	} else {
		remoteLTR, ok := remotePolicy.RetentionPolicy.(*armrecoveryservicesbackup.LongTermRetentionPolicy)
		if !ok {
			return 0, fmt.Errorf("%s: remote retention policy is not LongTermRetentionPolicy", label)
		}
		_, retDiff, _, _, err := azure_engine.DiffVMRetention(vm, remoteLTR)
		if err != nil {
			return 0, fmt.Errorf("%s: retention diff failed: %w", label, err)
		}
		_, schedDiff, _, _, err := azure_engine.DiffVMSchedule(vm, remotePolicy.SchedulePolicy)
		if err != nil {
			return 0, fmt.Errorf("%s: schedule diff failed: %w", label, err)
		}
		_, typeDiff, typeOverwrite := azure_engine.DiffVMPolicyType(vm, remotePolicy.PolicyType)
		if len(typeDiff) > 0 {
			return 0, fmt.Errorf(
				"%s: cannot change policy %q from %v to %v — Azure does not support updating an existing policy's type in place; create a new policy under a different metadata.name instead",
				label, vm.PolicyName, typeOverwrite["policyType"], typeDiff["policyType"],
			)
		}
		if len(retDiff) > 0 || len(schedDiff) > 0 {
			policyNeedsApply = true
		}
	}

	if policyNeedsApply {
		if dryRun {
			fmt.Printf("  [dry-run] would upsert policy %q\n", vm.PolicyName)
		} else {
			if err := client.CreateOrUpdateBackupPolicyVM(ctx, vm); err != nil {
				return 0, fmt.Errorf("%s: failed to upsert policy definition: %w", label, err)
			}
			fmt.Println(colorGreen(fmt.Sprintf("  ✔ Upserted policy %q", vm.PolicyName)))
			applied++
		}
	}

	// --- 2. Protected item (VM assignment to the policy) ---
	remote, err := client.GetBackupPolicyVM(ctx, vm.ResourceGroup, vm.VaultName, vm.ResourceName)
	if err != nil {
		return applied, fmt.Errorf("%s: failed to get protected item state: %w", label, err)
	}

	if remote == nil {
		if dryRun {
			fmt.Printf("  [dry-run] would assign policy %q to %s\n", vm.PolicyName, label)
			return applied, nil
		}
		if err := client.ApplyVM(ctx, vm); err != nil {
			return applied, fmt.Errorf("%s: apply failed: %w", label, err)
		}
		fmt.Println(colorGreen(fmt.Sprintf("✔ Applied policy %q to %s", vm.PolicyName, label)))
		return applied + 1, nil
	}

	_, itemDiff, _, _, err := azure_engine.DiffVMPolicy(vm, *remote)
	if err != nil {
		return applied, fmt.Errorf("%s: protected item diff failed: %w", label, err)
	}

	if len(itemDiff) == 0 {
		fmt.Println(colorGreen(fmt.Sprintf("✔ %s — protected item already assigned correctly", label)))
		return applied, nil
	}

	if vm.Spec.PolicyType == "Standard" {
		policyType, err := client.CurrentVMPolicyType(ctx, vm.Resource.Vault.ResourceGroup, vm.VaultName, remote)
		if err != nil {
			return applied, fmt.Errorf("%s: failed to check current policy type: %w", label, err)
		}
		if policyType == armrecoveryservicesbackup.IAASVMPolicyTypeV2 {
			return applied, fmt.Errorf(
				"%s: cannot assign a Standard policy — this VM is currently protected by an Enhanced (V2) policy, and Azure does not support downgrading VM protection from Enhanced to Standard",
				label,
			)
		}
	}

	if dryRun {
		fmt.Printf("  [dry-run] would assign policy %q to %s\n", vm.PolicyName, label)
		return applied, nil
	}

	if err := client.ApplyVM(ctx, vm); err != nil {
		return applied, fmt.Errorf("%s: apply failed: %w", label, err)
	}

	fmt.Println(colorGreen(fmt.Sprintf("✔ Applied policy %q to %s", vm.PolicyName, label)))
	return applied + 1, nil
}

func applyAzureDisk(ctx context.Context, client *azure_external.Client, disk azure.ResolvedDisk, dryRun bool) (int, error) {
	label := disk.ResourceGroup + "/" + disk.ResourceName
	applied := 0

	// --- 1. Check disk exists ---
	existingDisk, err := client.GetDisk(ctx, disk.ResourceGroup, disk.ResourceName)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get disk: %w", label, err)
	}
	if existingDisk == nil {
		return 0, fmt.Errorf("%s: disk does not exist", label)
	}

	// --- 2. Policy definition (retention / schedule in the vault) ---
	remotePolicy, err := client.GetBackupPolicyDefinitionDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to get policy definition: %w", label, err)
	}

	policyNeedsApply := false
	if remotePolicy == nil {
		fmt.Printf("  policy %q does not exist in vault — will create\n", disk.PolicyName)
		policyNeedsApply = true
	} else {
		_, retDiff, _, _, err := azure_engine.DiffDiskRetention(disk, remotePolicy)
		if err != nil {
			return 0, fmt.Errorf("%s: retention diff failed: %w", label, err)
		}
		if len(retDiff) > 0 {
			policyNeedsApply = true
		}
	}

	if policyNeedsApply {
		if dryRun {
			fmt.Printf("  [dry-run] would upsert policy %q\n", disk.PolicyName)
		} else {
			if err := client.CreateOrUpdateBackupPolicyDisk(ctx, disk); err != nil {
				return 0, fmt.Errorf("%s: failed to upsert policy definition: %w", label, err)
			}
			fmt.Println(colorGreen(fmt.Sprintf("  ✔ Upserted policy %q", disk.PolicyName)))
			applied++
		}
	}

	// --- 3. Backup instance (disk assignment to the policy) ---
	remote, err := client.GetBackupPolicyDisk(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.ResourceName)
	if err != nil {
		return applied, fmt.Errorf("%s: failed to get backup instance state: %w", label, err)
	}

	if remote == nil {
		if dryRun {
			fmt.Printf("  [dry-run] would register disk %q as backup instance under policy %q\n", disk.ResourceName, disk.PolicyName)
			return applied, nil
		}
		if err := client.ApplyDisk(ctx, disk); err != nil {
			return applied, fmt.Errorf("%s: apply failed: %w", label, err)
		}
		fmt.Println(colorGreen(fmt.Sprintf("✔ Applied policy %q to %s", disk.PolicyName, label)))
		return applied + 1, nil
	}

	_, itemDiff, _, _, err := azure_engine.DiffDiskPolicy(disk, *remote)
	if err != nil {
		return applied, fmt.Errorf("%s: backup instance diff failed: %w", label, err)
	}

	if len(itemDiff) == 0 {
		fmt.Println(colorGreen(fmt.Sprintf("✔ %s — backup instance already assigned correctly", label)))
		return applied, nil
	}

	if dryRun {
		fmt.Printf("  [dry-run] would assign policy %q to %s\n", disk.PolicyName, label)
		return applied, nil
	}

	if err := client.ApplyDisk(ctx, disk); err != nil {
		return applied, fmt.Errorf("%s: apply failed: %w", label, err)
	}

	fmt.Println(colorGreen(fmt.Sprintf("✔ Applied policy %q to %s", disk.PolicyName, label)))
	return applied + 1, nil
}
