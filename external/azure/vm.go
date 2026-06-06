package azure_external

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models"
)

// GetBackupPolicyVM retrieves the current backup configuration for an Azure VM
// from its Recovery Services Vault protected-item record.
func (c *Client) GetBackupPolicyVM(
	ctx context.Context,
	resourceGroup, vaultName, vmName string,
) (armrecoveryservicesbackup.ProtectedItemsClientGetResponse, error) {
	var result armrecoveryservicesbackup.ProtectedItemsClientGetResponse

	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return result, fmt.Errorf("failed to create protected items client: %w", err)
	}

	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + resourceGroup + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + resourceGroup + ";" + vmName

	result, err = client.Get(ctx, vaultName, resourceGroup, "Azure", containerName, protectedItemName, nil)
	if err != nil {
		return result, fmt.Errorf("failed to get backup config for VM %q in vault %q: %w", vmName, vaultName, err)
	}
	return result, nil
}

// ApplyVM applies the backup policy from a ResolvedTarget to an Azure VM.
func (c *Client) ApplyVM(ctx context.Context, rt azure_models.ResolvedVM) error {
	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create protected items client: %w", err)
	}

	rg := rt.ResourceGroup
	vmName := rt.ResourceName
	vaultName := rt.VaultName
	// Can we assume that vaultRG is the same as the VM's RG?
	// NOTE: Needed? vaultRG := rt.Policy.Spec.Vault.ResourceGroup
	policyName := rt.PolicyName
	diskSelection := rt.Spec.DiskSelection

	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + rg + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + rg + ";" + vmName

	var props armrecoveryservicesbackup.ProtectedItemResource
	switch {
	case diskSelection == nil || diskSelection.Mode == "all":
		// No disk filter — back up everything including future disks
		props = buildVMProtectedItem(policyName, make([]*int32, 0), to.Ptr(false))
	case diskSelection.Mode == "include":
		props = buildVMProtectedItem(policyName, toLUNPtrs(diskSelection.LUNs), to.Ptr(false))
	case diskSelection.Mode == "exclude":
		props = buildVMProtectedItem(policyName, toLUNPtrs(diskSelection.LUNs), to.Ptr(true))
	default:
		return fmt.Errorf("invalid disk_selection.mode %q: expected all, include, or exclude", diskSelection.Mode)
	}

	_, err = client.CreateOrUpdate(ctx, vaultName, vaultName, "Azure", containerName, protectedItemName, props, nil)
	if err != nil {
		return fmt.Errorf("failed to apply backup policy for VM %q: %w", vmName, err)
	}
	return nil
}

func buildVMProtectedItem(policyName string, lunList []*int32, isInclusionList *bool) armrecoveryservicesbackup.ProtectedItemResource {
	return armrecoveryservicesbackup.ProtectedItemResource{
		Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
			ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
				DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
					DiskLunList:     lunList,
					IsInclusionList: isInclusionList,
				},
			},
			PolicyName: to.Ptr(policyName),
		},
	}
}

func toLUNPtrs(luns []int32) []*int32 {
	ptrs := make([]*int32, len(luns))
	for i, l := range luns {
		ptrs[i] = to.Ptr(l)
	}
	return ptrs
}
