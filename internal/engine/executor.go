package engine

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
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

func ApplyAzureVM(cred *azidentity.DefaultAzureCredential, subscriptionID string, resourceGroup string, vmName string, vaultName string, policyID string, diskLUNs []int32, includeFutureDisks bool) error {
	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create protected items client: %w", err)
	}

	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + resourceGroup + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + resourceGroup + ";" + vmName

	var properties armrecoveryservicesbackup.ProtectedItemResource

	if (len(diskLUNs) == 0) && (includeFutureDisks) {
		// Pass, properties should not be initialized
	} else if (len(diskLUNs) > 0) && (includeFutureDisks) {
		lunPointers := createLUNPointers(diskLUNs)

		properties = armrecoveryservicesbackup.ProtectedItemResource{
			Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
				ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
					DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
						DiskLunList:     lunPointers,
						IsInclusionList: to.Ptr(false),
					},
				},
				PolicyID: to.Ptr(policyID),
			},
		}
	} else if (len(diskLUNs) > 0) && (!includeFutureDisks) {
		lunPointers := createLUNPointers(diskLUNs)

		properties = armrecoveryservicesbackup.ProtectedItemResource{
			Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
				ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
					DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
						DiskLunList:     lunPointers,
						IsInclusionList: to.Ptr(true),
					},
				},
				PolicyID: to.Ptr(policyID),
			},
		}
	} else {
		return fmt.Errorf("Impossible scenario - we're in trouble..")
	}
	_, err = client.CreateOrUpdate(
		context.Background(),
		vaultName,
		resourceGroup,
		"Azure",
		containerName,
		protectedItemName,
		properties,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to update backup policy for VM %q: %w", vmName, err)
	}

	return nil
}

func createLUNPointers(diskLUNs []int32) []*int32 {
	lunPointers := make([]*int32, len(diskLUNs))
	for i, lun := range diskLUNs {
		lunPointers[i] = to.Ptr(lun)
	}
	return lunPointers
}

// Gets the backupPolicy that is applied to a VM from Azure
// Parameters:
// returns the clientResponse from Azure
func GetBackupPolicyVM(cred *azidentity.DefaultAzureCredential, resourceGroup string,
	vaultName string,
	backupPolicy string,
	provider string,
	vmName string,
	subscriptionID string) (armrecoveryservicesbackup.ProtectedItemsClientGetResponse, error) {
	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + resourceGroup + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + resourceGroup + ";" + vmName

	var result armrecoveryservicesbackup.ProtectedItemsClientGetResponse

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

func GetBackupPolicyDB(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	resourceGroup string,
	serverName string,
	dbName string,
) (armsql.BackupShortTermRetentionPoliciesClientGetResponse, armsql.LongTermRetentionPolicy, error) {
	// Short-term retention (PITR): controls retention days and differential backup interval
	var shortTermClient armsql.BackupShortTermRetentionPoliciesClientGetResponse
	var longTermClient armsql.LongTermRetentionPolicy
	var err error

	shortTermClient, err = armsql.NewBackupShortTermRetentionPoliciesClient(subscriptionID, cred, nil)
	if err != nil {
		return shortTermClient, longTermClient, fmt.Errorf("failed to create short-term retention client: %w", err)
	}

	shortTermResult, err := shortTermClient.Get(
		context.Background(),
		resourceGroup,
		serverName,
		dbName,
		armsql.ShortTermRetentionPolicyNameDefault,
		nil,
	)
	if err != nil {
		return shortTermClient, longTermClient, fmt.Errorf("failed to get short-term retention policy for DB %q on server %q: %w", dbName, serverName, err)
	}

	// Long-term retention (LTR): controls weekly/monthly/yearly backup retention
	longTermClient, err = armsql.NewLongTermRetentionPoliciesClient(subscriptionID, cred, nil)
	if err != nil {
		return shortTermClient, longTermClient, fmt.Errorf("failed to create long-term retention client: %w", err)
	}

	longTermResult, err := longTermClient.Get(
		context.Background(),
		resourceGroup,
		serverName,
		dbName,
		armsql.LongTermRetentionPolicyNameDefault,
		nil,
	)
	if err != nil {
		return shortTermClient, longTermClient, fmt.Errorf("failed to get long-term retention policy for DB %q on server %q: %w", dbName, serverName, err)
	}

	return &shortTermResult.BackupShortTermRetentionPolicy, &longTermResult.LongTermRetentionPolicy, nil
}
