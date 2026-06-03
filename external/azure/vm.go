package azure_external

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
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
func (c *Client) ApplyVM(ctx context.Context, rt models.ResolvedTarget) error {
	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create protected items client: %w", err)
	}

	rg := rt.Target.ResourceGroup
	vmName := rt.Target.ResourceName
	vaultName := rt.Policy.Spec.Vault.Name
	vaultRG := rt.Policy.Spec.Vault.ResourceGroup
	policyName := rt.Policy.Metadata.Name
	diskLUNs := rt.Target.DiskSelection
	includeFuture := rt.Target.IncludeFutureDisks

	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + rg + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + rg + ";" + vmName

	var props armrecoveryservicesbackup.ProtectedItemResource
	switch {
	case len(diskLUNs) == 0 && includeFuture:
		props = buildVMProtectedItem(policyName, make([]*int32, 0), to.Ptr(false))
	case len(diskLUNs) > 0 && includeFuture:
		props = buildVMProtectedItem(policyName, toLUNPtrs(diskLUNs), to.Ptr(false))
	case len(diskLUNs) > 0 && !includeFuture:
		props = buildVMProtectedItem(policyName, toLUNPtrs(diskLUNs), to.Ptr(true))
	default:
		return fmt.Errorf("invalid disk selection: no disks selected and include_future_disks=false")
	}

	_, err = client.CreateOrUpdate(ctx, vaultName, vaultRG, "Azure", containerName, protectedItemName, props, nil)
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
