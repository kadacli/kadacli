package azure_external

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dataprotection/armdataprotection"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// GetBackupPolicyDisk retrieves the current backup instance configuration for an Azure Managed Disk.
func (c *Client) GetBackupPolicyDisk(
	ctx context.Context,
	resourceGroup, vaultName, diskName string,
) (*armdataprotection.BackupInstanceResource, error) {
	client, err := armdataprotection.NewBackupInstancesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup instances client: %w", err)
	}

	instanceName := diskName
	result, err := client.Get(ctx, resourceGroup, vaultName, instanceName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup instance for disk %q in vault %q: %w", diskName, vaultName, err)
	}
	return &result.BackupInstanceResource, nil
}

// ApplyDisk assigns a backup policy to an Azure Managed Disk via Backup Vault.
func (c *Client) ApplyDisk(ctx context.Context, rt models.ResolvedTarget) error {
	client, err := armdataprotection.NewBackupInstancesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create backup instances client: %w", err)
	}

	rg := rt.Target.ResourceGroup
	diskName := rt.Target.ResourceName
	vaultName := rt.Policy.Spec.Vault.Name
	vaultRG := rt.Policy.Spec.Vault.ResourceGroup
	policyID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/backupVaults/%s/backupPolicies/%s",
		c.SubscriptionID, vaultRG, vaultName, rt.Policy.Metadata.Name,
	)
	diskID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/disks/%s",
		c.SubscriptionID, rg, diskName,
	)
	snapshotRG := fmt.Sprintf(
		"%s-%s-%s",
		rt.Policy.Spec.SnapshotResourceGroupPrefix,
		rg,
		rt.Policy.Spec.SnapshotResourceGroupSuffix,
	)

	instance := armdataprotection.BackupInstanceResource{
		Properties: &armdataprotection.BackupInstance{
			FriendlyName: to.Ptr(diskName),
			DataSourceInfo: &armdataprotection.Datasource{
				DatasourceType:   to.Ptr("Microsoft.Compute/disks"),
				ObjectType:       to.Ptr("Datasource"),
				ResourceID:       to.Ptr(diskID),
				ResourceName:     to.Ptr(diskName),
				ResourceType:     to.Ptr("Microsoft.Compute/disks"),
				ResourceLocation: to.Ptr(""),
				ResourceURI:      to.Ptr(diskID),
			},
			PolicyInfo: &armdataprotection.PolicyInfo{
				PolicyID: to.Ptr(policyID),
				PolicyParameters: &armdataprotection.PolicyParameters{
					DataStoreParametersList: []armdataprotection.DataStoreParametersClassification{
						&armdataprotection.AzureOperationalStoreParameters{
							DataStoreType:   to.Ptr(armdataprotection.DataStoreTypesOperationalStore),
							ObjectType:      to.Ptr("AzureOperationalStoreParameters"),
							ResourceGroupID: to.Ptr(snapshotRG),
						},
					},
				},
			},
			ObjectType: to.Ptr("BackupInstance"),
		},
	}

	poller, err := client.BeginCreateOrUpdate(ctx, vaultRG, vaultName, diskName, instance, nil)
	if err != nil {
		return fmt.Errorf("failed to begin applying backup policy for disk %q: %w", diskName, err)
	}
	if _, err := poller.PollUntilDone(ctx, nil); err != nil {
		return fmt.Errorf("failed to apply backup policy for disk %q: %w", diskName, err)
	}
	return nil
}

// CreateOrUpdateBackupPolicyDisk creates or updates a disk backup policy in a Backup Vault.
func (c *Client) CreateOrUpdateBackupPolicyDisk(
	ctx context.Context,
	cfg azure.AzureRecoveryDiskPolicy,
) (string, error) {
	client, err := armdataprotection.NewBackupPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create backup policies client: %w", err)
	}

	// Interval is stored as hours, Azure expects an ISO 8601 duration string.
	intervalDuration := fmt.Sprintf("PT%dH", cfg.BackupIntervalHours)
	retentionDuration := fmt.Sprintf("P%dD", cfg.RetentionDays)
	triggerTime := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	policy := armdataprotection.BaseBackupPolicyResource{
		Properties: &armdataprotection.BackupPolicy{
			ObjectType:      to.Ptr("BackupPolicy"),
			DatasourceTypes: []*string{to.Ptr("Microsoft.Compute/disks")},
			PolicyRules: []armdataprotection.BasePolicyRuleClassification{
				&armdataprotection.AzureBackupRule{
					Name:       to.Ptr("BackupHourly"),
					ObjectType: to.Ptr("AzureBackupRule"),
					BackupParameters: &armdataprotection.AzureBackupParams{
						ObjectType: to.Ptr("AzureBackupParams"),
						BackupType: to.Ptr("Incremental"),
					},
					DataStore: &armdataprotection.DataStoreInfoBase{
						DataStoreType: to.Ptr(armdataprotection.DataStoreTypesOperationalStore),
						ObjectType:    to.Ptr("DataStoreInfoBase"),
					},
					Trigger: &armdataprotection.ScheduleBasedTriggerContext{
						ObjectType: to.Ptr("ScheduleBasedTriggerContext"),
						Schedule: &armdataprotection.BackupSchedule{
							RepeatingTimeIntervals: []*string{
								to.Ptr(fmt.Sprintf("R/2024-01-01T00:00:00Z/%s", intervalDuration)),
							},
						},
						TaggingCriteria: []*armdataprotection.TaggingCriteria{
							{
								IsDefault:       to.Ptr(true),
								TaggingPriority: to.Ptr[int64](99),
								TagInfo: &armdataprotection.RetentionTag{
									TagName: to.Ptr("Default"),
								},
							},
						},
					},
				},
				&armdataprotection.AzureRetentionRule{
					Name:       to.Ptr("Default"),
					ObjectType: to.Ptr("AzureRetentionRule"),
					IsDefault:  to.Ptr(true),
					Lifecycles: []*armdataprotection.SourceLifeCycle{
						{
							DeleteAfter: &armdataprotection.AbsoluteDeleteOption{
								ObjectType: to.Ptr("AbsoluteDeleteOption"),
								Duration:   to.Ptr(retentionDuration),
							},
							SourceDataStore: &armdataprotection.DataStoreInfoBase{
								DataStoreType: to.Ptr(armdataprotection.DataStoreTypesOperationalStore),
								ObjectType:    to.Ptr("DataStoreInfoBase"),
							},
						},
					},
				},
			},
		},
	}
	_ = triggerTime

	resp, err := client.CreateOrUpdate(ctx, cfg.ResourceGroup, cfg.VaultName, cfg.PolicyName, policy, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create/update disk backup policy %q: %w", cfg.PolicyName, err)
	}
	if resp.ID == nil {
		return "", fmt.Errorf("disk backup policy %q created but response contained no ID", cfg.PolicyName)
	}
	return *resp.ID, nil
}

// GetVMDataDiskLUNs returns the LUN numbers of all data disks attached to the VM.
func GetVMDataDiskLUNs(cred *azidentity.DefaultAzureCredential, subscriptionID, resourceGroup, vmName string) ([]int32, error) {
	client, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %w", err)
	}
	vm, err := client.Get(context.Background(), resourceGroup, vmName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM %q: %w", vmName, err)
	}
	var luns []int32
	if vm.Properties != nil && vm.Properties.StorageProfile != nil {
		for _, disk := range vm.Properties.StorageProfile.DataDisks {
			if disk.Lun != nil {
				luns = append(luns, *disk.Lun)
			}
		}
	}
	return luns, nil
}
