package azure_external

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dataprotection/armdataprotection"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure/policies"
)

// Looks for the disk in Azure usin the armcompute module
func (c *Client) GetDisk(ctx context.Context, resourceGroup, diskName string) (*armcompute.Disk, error) {
	computeClient, err := armcompute.NewDisksClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create compute client: %w", err)
	}
	resp, err := computeClient.Get(ctx, resourceGroup, diskName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to get disk %q: %w", diskName, err)
	}
	return &resp.Disk, nil
}

// GetBackupPolicyDisk retrieves the current backup configuration for an Azure Disk
// from its Backup Vault backup instance record.
func (c *Client) GetBackupPolicyDisk(
	ctx context.Context,
	resourceGroup, vaultName, diskName string,
) (*armdataprotection.BackupInstancesClientGetResponse, error) {
	instancesClient, err := armdataprotection.NewBackupInstancesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create backup instances client: %w", err)
	}

	result, err := instancesClient.Get(ctx, resourceGroup, vaultName, diskName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to get backup config for disk %q in vault %q: %w", diskName, vaultName, err)
	}
	return &result, nil
}

// GetBackupPolicyDefinitionDisk fetches the named policy definition from the Backup Vault.
// Returns (nil, nil) when the policy does not exist yet (HTTP 404).
func (c *Client) GetBackupPolicyDefinitionDisk(
	ctx context.Context,
	vaultRG, vaultName, policyName string,
) (*armdataprotection.BackupPolicy, error) {
	client, err := armdataprotection.NewBackupPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create backup policies client: %w", err)
	}

	resp, err := client.Get(ctx, vaultRG, vaultName, policyName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to get policy %q from vault %q: %w", policyName, vaultName, err)
	}

	policy, ok := resp.Properties.(*armdataprotection.BackupPolicy)
	if !ok {
		return nil, fmt.Errorf("Policy %q is not a BackupPolicy (got %T)", policyName, resp.Properties)
	}
	return policy, nil
}

// CreateOrUpdateBackupPolicyDisk upserts the backup policy definition in the Backup Vault
// using the spec from the resolved disk.
func (c *Client) CreateOrUpdateBackupPolicyDisk(ctx context.Context, disk azure.ResolvedDisk) error {
	client, err := armdataprotection.NewBackupPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("Failed to create backup policies client: %w", err)
	}

	// Disk backup policies cannot be updated — delete first if exists
	_, delErr := client.Delete(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName, nil)
	if delErr != nil {
		var respErr *azcore.ResponseError
		if !errors.As(delErr, &respErr) || respErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Failed to delete existing policy %q before recreating: %w", disk.PolicyName, delErr)
		}
		// 404 is fine — policy didn't exist yet
	}

	policy := buildDiskBackupPolicy(disk)
	_, err = client.CreateOrUpdate(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName, policy, nil)
	if err != nil {
		return fmt.Errorf("Failed to upsert policy %q in vault %q: %w", disk.PolicyName, disk.VaultName, err)
	}
	return nil
}

func buildDiskBackupPolicy(disk azure.ResolvedDisk) armdataprotection.BaseBackupPolicyResource {
	spec := disk.Spec

	var policyRules []armdataprotection.BasePolicyRuleClassification

	// Backup rule name based on frequency
	backupRuleName := "BackupDaily"
	if spec.Schedule.Frequency == "hourly" {
		backupRuleName = "BackupHourly"
	}

	// Build tagging criteria — one entry per retention rule plus Default
	taggingCriteria := []*armdataprotection.TaggingCriteria{
		{
			IsDefault:       to.Ptr(true),
			TaggingPriority: to.Ptr(int64(99)),
			TagInfo: &armdataprotection.RetentionTag{
				ID:      to.Ptr("Default_"),
				TagName: to.Ptr("Default"),
			},
		},
	}

	// Backup rule (schedule)
	policyRules = append(policyRules, &armdataprotection.AzureBackupRule{
		Name:       to.Ptr(backupRuleName),
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
				RepeatingTimeIntervals: []*string{to.Ptr(buildDiskScheduleInterval(spec.Schedule))},
			},
			TaggingCriteria: taggingCriteria,
		},
	})

	// Default retention rule (always required)
	defaultDuration := "P7D" // fallback
	for _, r := range spec.Retention.RetentionRules {
		if spec.Retention.Default {
			switch r.RetentionRule {
			case "daily":
				defaultDuration = fmt.Sprintf("P%dD", r.RetentionDuration)
			case "weekly":
				defaultDuration = fmt.Sprintf("P%dW", r.RetentionDuration)
			}
		}
	}

	policyRules = append(policyRules, &armdataprotection.AzureRetentionRule{
		Name:       to.Ptr("Default"),
		ObjectType: to.Ptr("AzureRetentionRule"),
		IsDefault:  to.Ptr(true),
		Lifecycles: []*armdataprotection.SourceLifeCycle{
			{
				DeleteAfter: &armdataprotection.AbsoluteDeleteOption{
					ObjectType: to.Ptr("AbsoluteDeleteOption"),
					Duration:   to.Ptr(defaultDuration),
				},
				SourceDataStore: &armdataprotection.DataStoreInfoBase{
					DataStoreType: to.Ptr(armdataprotection.DataStoreTypesOperationalStore),
					ObjectType:    to.Ptr("DataStoreInfoBase"),
				},
			},
		},
	})

	// Additional retention rules from spec
	for _, r := range spec.Retention.RetentionRules {
		var duration string
		switch r.RetentionRule {
		case "daily":
			duration = fmt.Sprintf("P%dD", r.RetentionDuration)
		case "weekly":
			duration = fmt.Sprintf("P%dW", r.RetentionDuration)
		}

		policyRules = append(policyRules, &armdataprotection.AzureRetentionRule{
			Name:       to.Ptr(r.RetentionRule),
			ObjectType: to.Ptr("AzureRetentionRule"),
			IsDefault:  to.Ptr(false),
			Lifecycles: []*armdataprotection.SourceLifeCycle{
				{
					DeleteAfter: &armdataprotection.AbsoluteDeleteOption{
						ObjectType: to.Ptr("AbsoluteDeleteOption"),
						Duration:   to.Ptr(duration),
					},
					SourceDataStore: &armdataprotection.DataStoreInfoBase{
						DataStoreType: to.Ptr(armdataprotection.DataStoreTypesOperationalStore),
						ObjectType:    to.Ptr("DataStoreInfoBase"),
					},
				},
			},
		})
	}

	return armdataprotection.BaseBackupPolicyResource{
		Properties: &armdataprotection.BackupPolicy{
			ObjectType:      to.Ptr("BackupPolicy"),
			DatasourceTypes: []*string{to.Ptr("Microsoft.Compute/disks")},
			PolicyRules:     policyRules,
		},
	}
}

// ApplyDisk registers the disk as a backup instance under the named policy in the Backup Vault.
func (c *Client) ApplyDisk(ctx context.Context, disk azure.ResolvedDisk) error {
	// Look up disk location from Azure
	computeClient, err := armcompute.NewDisksClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("Failed to create compute client: %w", err)
	}
	diskResp, err := computeClient.Get(ctx, disk.ResourceGroup, disk.ResourceName, nil)
	if err != nil {
		return fmt.Errorf("Failed to get disk %q to determine location: %w", disk.ResourceName, err)
	}
	location := *diskResp.Location

	diskResourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/disks/%s",
		c.SubscriptionID, disk.ResourceGroup, disk.ResourceName)
	policyID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataProtection/backupVaults/%s/backupPolicies/%s",
		c.SubscriptionID, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName)

	// Default snapshot resource group to the disk's resource group if not specified
	snapshotRG := disk.ResourceGroup
	if disk.Spec.SnapshotResourceGroup != "" {
		snapshotRG = disk.Spec.SnapshotResourceGroup
	}
	snapshotRGID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s",
		c.SubscriptionID, snapshotRG)

	instance := armdataprotection.BackupInstanceResource{
		Properties: &armdataprotection.BackupInstance{
			ObjectType:   to.Ptr("BackupInstance"),
			FriendlyName: to.Ptr(disk.ResourceName),
			DataSourceInfo: &armdataprotection.Datasource{
				ObjectType:       to.Ptr("Datasource"),
				ResourceID:       to.Ptr(diskResourceID),
				ResourceName:     to.Ptr(disk.ResourceName),
				ResourceType:     to.Ptr("Microsoft.Compute/disks"),
				ResourceLocation: to.Ptr(location),
				DatasourceType:   to.Ptr("Microsoft.Compute/disks"),
			},
			PolicyInfo: &armdataprotection.PolicyInfo{
				PolicyID: to.Ptr(policyID),
				PolicyParameters: &armdataprotection.PolicyParameters{
					DataStoreParametersList: []armdataprotection.DataStoreParametersClassification{
						&armdataprotection.AzureOperationalStoreParameters{
							ObjectType:      to.Ptr("AzureOperationalStoreParameters"),
							DataStoreType:   to.Ptr(armdataprotection.DataStoreTypesOperationalStore),
							ResourceGroupID: to.Ptr(snapshotRGID),
						},
					},
				},
			},
		},
	}

	instancesClient, err := armdataprotection.NewBackupInstancesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("Failed to create backup instances client: %w", err)
	}

	poller, err := instancesClient.BeginCreateOrUpdate(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.ResourceName, instance, nil)
	if err != nil {
		return fmt.Errorf("Failed to begin applying backup policy for disk %q: %w", disk.ResourceName, err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("Failed to apply backup policy for disk %q: %w", disk.ResourceName, err)
	}
	return nil
}

func buildDiskScheduleInterval(sched policies.AzureDiskSchedule) string {
	switch sched.Frequency {
	case "hourly":
		return fmt.Sprintf("R/2023-01-01T00:00:00Z/PT%dH", sched.Interval)
	case "daily":
		return fmt.Sprintf("R/2023-01-01T%s:00Z/P1D", sched.Time)
	default:
		return "R/2023-01-01T00:00:00Z/PT4H"
	}
}
