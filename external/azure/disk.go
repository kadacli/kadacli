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

	policy := buildDiskBackupPolicy(disk)

	// Disk backup policies are immutable: Azure rejects CreateOrUpdate on an
	// existing policy with UserErrorBMSUpdatePolicyNotSupported ("Update of
	// existing policy is not supported. Please create a new policy."), so the
	// only way to change one is to delete and recreate it. That in turn is
	// refused with UserErrorDppPolicyObjectInUse while any backup instance is
	// protected by it — which makes an applied policy effectively read-only.
	// Both were confirmed against the live API; don't replace this with a
	// plain CreateOrUpdate.
	_, delErr := client.Delete(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName, nil)
	if delErr != nil {
		var respErr *azcore.ResponseError
		if errors.As(delErr, &respErr) && respErr.ErrorCode == "UserErrorDppPolicyObjectInUse" {
			return fmt.Errorf(
				"Policy %q is in use by a backup instance and its definition has changed. Azure does not support "+
					"updating a disk backup policy in place, and refuses to delete one that is protecting a disk. "+
					"Give the policy a new metadata.name and re-apply — the disk moves to the new policy and the old "+
					"one can then be deleted from vault %q",
				disk.PolicyName, disk.VaultName,
			)
		}
		if !errors.As(delErr, &respErr) || respErr.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Failed to delete existing policy %q before recreating: %w", disk.PolicyName, delErr)
		}
		// 404 is fine — policy didn't exist yet
	}

	_, err = client.CreateOrUpdate(ctx, disk.Resource.Vault.ResourceGroup, disk.VaultName, disk.PolicyName, policy, nil)
	if err != nil {
		return fmt.Errorf("Failed to create policy %q in vault %q: %w", disk.PolicyName, disk.VaultName, err)
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

	// Build tagging criteria — one entry per retention rule plus Default.
	// Azure applies a retention rule only to backups carrying a tag of the same
	// name, so a rule without a criterion here is inert: it appears in the
	// policy and never retains anything.
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

	for _, r := range spec.Retention.RetentionRules {
		tag := r.TagName()

		// Which backup earns the tag: the first of each day, or the first of
		// each week. Lower tagging priority wins over the Default's 99.
		marker := armdataprotection.AbsoluteMarkerFirstOfDay
		priority := int64(25)
		if tag == "Weekly" {
			marker = armdataprotection.AbsoluteMarkerFirstOfWeek
			priority = int64(20)
		}

		taggingCriteria = append(taggingCriteria, &armdataprotection.TaggingCriteria{
			IsDefault:       to.Ptr(false),
			TaggingPriority: to.Ptr(priority),
			TagInfo: &armdataprotection.RetentionTag{
				ID:      to.Ptr(tag + "_"),
				TagName: to.Ptr(tag),
			},
			Criteria: []armdataprotection.BackupCriteriaClassification{
				&armdataprotection.ScheduleBasedBackupCriteria{
					ObjectType:       to.Ptr("ScheduleBasedBackupCriteria"),
					AbsoluteCriteria: []*armdataprotection.AbsoluteMarker{to.Ptr(marker)},
				},
			},
		})
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
				// Without this Azure runs the schedule in UTC regardless of the
				// time in the interval string. Windows time zone ID, e.g.
				// "W. Europe Standard Time".
				TimeZone: to.Ptr(disk.Timezone),
			},
			TaggingCriteria: taggingCriteria,
		},
	})

	// Default retention rule (always required)
	defaultDuration := spec.Retention.DefaultISODuration()

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
		duration := r.ISODuration()

		policyRules = append(policyRules, &armdataprotection.AzureRetentionRule{
			// Must match the tag name above — Azure pairs rules to tags by name.
			Name:       to.Ptr(r.TagName()),
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

	// Defaults to the disk's own resource group when spec.snapshot_resource_group
	// is unset. Shared with the differ so the two cannot disagree.
	snapshotRGID := disk.SnapshotResourceGroupID()

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
