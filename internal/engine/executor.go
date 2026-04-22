package engine

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
)

func Authenticate() (*azidentity.DefaultAzureCredential, error) {
	// TODO: Authenticate towards Azure with GO SDK
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to obtain credentials %w:", err)
	}
	return cred, nil
}

func GetAzureVault(cred *azidentity.DefaultAzureCredential, subscriptionID string, resourceGroup string, vaultName string) (*armrecoveryservices.Vault, error) {
	// TODO: Gets Azure vault from Go SDK(maybe id?)
	client, err := armrecoveryservices.NewVaultsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vaults client: %w", err)
	}

	result, err := client.Get(context.Background(), resourceGroup, vaultName, nil)
	if err != nil {
		return nil, fmt.Errorf("vault %q not found in resource group %q: %w", vaultName, resourceGroup, err)
	}

	return &result.Vault, nil
}

func GetAzureBackupPolicy(cred *azidentity.DefaultAzureCredential, subscriptionID string, resourceGroup string, vaultName string, policyName string) (string, error) {
	// TODO: Check validity of backup_policy in Azure (that it exists?) get an string or id back, and apply it to target resource name

	client, err := armrecoveryservicesbackup.NewProtectionPoliciesClient(subscriptionID, cred, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create protection policies client: %w", err)
	}

	result, err := client.Get(context.Background(), vaultName, resourceGroup, policyName, nil)
	if err != nil {
		return "", fmt.Errorf("policy %q not found in vault %q: %w", policyName, vaultName, err)
	}

	if result.ID == nil {
		return "", fmt.Errorf("policy %q exists but has no ID", policyName)
	}

	return *result.ID, nil
}

func ApplyAzureVM(cred *azidentity.DefaultAzureCredential, subscriptionID string, resourceGroup string, vmName string, vaultName string, policyName string, diskLUNs []int32, includeFutureDisks bool) error {
	// TODO: Check validity of policy_subtype, disk_selection, and include_future_disks
	// and then implement
	// TODO: Future disks inclusion not done.

	// Validate vault exists
	if _, err := GetAzureVault(cred, subscriptionID, resourceGroup, vaultName); err != nil {
		return fmt.Errorf("invalid vault: %w", err)
	}

	// Validate policy exists and get its ID
	policyID, err := GetAzureBackupPolicy(cred, subscriptionID, resourceGroup, vaultName, policyName)
	if err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create protected items client: %w", err)
	}

	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + resourceGroup + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + resourceGroup + ";" + vmName

	// Build disk exclusion properties
	lunPointers := make([]*int32, len(diskLUNs))
	for i, lun := range diskLUNs {
		lunPointers[i] = to.Ptr(lun)
	}

	_, err = client.CreateOrUpdate(
		context.Background(),
		vaultName,
		resourceGroup,
		"Azure",
		containerName,
		protectedItemName,
		armrecoveryservicesbackup.ProtectedItemResource{
			Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
				ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
					DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
						DiskLunList:     lunPointers,
						IsInclusionList: to.Ptr(true),
					},
				},
				PolicyID: to.Ptr(policyID),
			},
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to update backup policy for VM %q: %w", vmName, err)
	}

	return nil
}

// Gets the backupPolicy that is applied to a VM from Azure
// Parameters:
// returns the clientResponse from Azure
func GetBackupPolicyVM(resourceGroup string,
	vaultName string,
	backupPolicy string,
	provider string,
	vmName string,
	subscriptionID string) (armrecoveryservicesbackup.ProtectedItemsClientGetResponse, error) {
	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + resourceGroup + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + resourceGroup + ";" + vmName

	var result armrecoveryservicesbackup.ProtectedItemsClientGetResponse

	// Authenticate
	cred, err := Authenticate()
	if err != nil {
		return result, err
	}

	switch provider {
	case "azure":
		client, err := armrecoveryservicesbackup.NewProtectedItemsClient(subscriptionID, cred, nil)
		if err != nil {
			return result, fmt.Errorf("failed to create protection policies client: %w", err)
		}

		result, err := client.Get(
			context.Background(),
			vaultName, // Recovery Services Vault name
			resourceGroup,
			"Azure",           // e.g., "Azure"
			containerName,     // e.g., "iaasvmcontainer;iaasvmcontainerv2;myRG;myVM"
			protectedItemName, // e.g., "vm;iaasvmcontainerv2;myRG;myVM"
			nil,
		)

		//result, err := client.Get(context.Background(), vaultName, resourceGroup, backupPolicy, nil)
		if err != nil {
			return result, fmt.Errorf("policy %q not found in vault %q: %w", backupPolicy, vaultName, err)
		}
		return result, nil

	case "aws":
		return result, fmt.Errorf("AWS is not supported by bpac a provider. Currently, only Azure is supported")
	default:
		return result, fmt.Errorf("Invalid provider set in Spec.StorageProvider.Provider")
	}

	return result, nil
}
