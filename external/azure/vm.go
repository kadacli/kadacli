package azure_external

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure/policies"
)

// GetVM looks up the VM in Azure using the armcompute module.
// Returns (nil, nil) when the VM does not exist (HTTP 404).
func (c *Client) GetVM(ctx context.Context, resourceGroup, vmName string) (*armcompute.VirtualMachine, error) {
	computeClient, err := armcompute.NewVirtualMachinesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}
	resp, err := computeClient.Get(ctx, resourceGroup, vmName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get VM %q: %w", vmName, err)
	}
	return &resp.VirtualMachine, nil
}

// GetBackupPolicyVM retrieves the current backup configuration for an Azure VM
// from its Recovery Services Vault protected-item record.
// Returns (nil, nil) when the VM has not yet been registered as a protected
// item in the vault (HTTP 404).
func (c *Client) GetBackupPolicyVM(
	ctx context.Context,
	resourceGroup, vaultName, vmName string,
) (*armrecoveryservicesbackup.ProtectedItemsClientGetResponse, error) {
	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create protected items client: %w", err)
	}

	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + resourceGroup + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + resourceGroup + ";" + vmName

	result, err := client.Get(ctx, vaultName, resourceGroup, "Azure", containerName, protectedItemName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get backup config for VM %q in vault %q: %w", vmName, vaultName, err)
	}
	return &result, nil
}

// IsVMBackupPaused reports whether the given protected item's backup is
// currently paused or stopped (e.g. "Stop backup with retain data" in the
// Portal). Azure's enable/update-protection call does not reliably resume a
// paused item, so callers should surface this to the user rather than
// attempt to apply through it.
func IsVMBackupPaused(remote *armrecoveryservicesbackup.ProtectedItemsClientGetResponse) bool {
	if remote == nil {
		return false
	}
	vmItem, ok := remote.Properties.(*armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem)
	if !ok || vmItem.ProtectionState == nil {
		return false
	}
	switch *vmItem.ProtectionState {
	case armrecoveryservicesbackup.ProtectionStateProtectionStopped, armrecoveryservicesbackup.ProtectionStateProtectionPaused:
		return true
	default:
		return false
	}
}

// CurrentVMPolicyType returns the PolicyType (V1/Standard or V2/Enhanced) of
// whatever policy the given protected item is currently assigned to in Azure.
// Returns "" if remote is nil (VM not yet protected) or the assigned policy
// can't be resolved.
func (c *Client) CurrentVMPolicyType(
	ctx context.Context,
	vaultRG, vaultName string,
	remote *armrecoveryservicesbackup.ProtectedItemsClientGetResponse,
) (armrecoveryservicesbackup.IAASVMPolicyType, error) {
	if remote == nil {
		return "", nil
	}
	vmItem, ok := remote.Properties.(*armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem)
	if !ok || vmItem.PolicyName == nil || *vmItem.PolicyName == "" {
		// No policy currently assigned (e.g. backup is paused/stopped) — nothing to conflict with.
		return "", nil
	}
	currentPolicy, err := c.GetBackupPolicyDefinitionVM(ctx, vaultRG, vaultName, *vmItem.PolicyName)
	if err != nil {
		return "", fmt.Errorf("failed to look up currently assigned policy %q: %w", *vmItem.PolicyName, err)
	}
	if currentPolicy == nil || currentPolicy.PolicyType == nil {
		return "", nil
	}
	return *currentPolicy.PolicyType, nil
}

// GetBackupPolicyDefinitionVM fetches the named policy definition from the vault.
// Returns (nil, nil) when the policy does not exist yet (HTTP 404).
func (c *Client) GetBackupPolicyDefinitionVM(
	ctx context.Context,
	vaultRG, vaultName, policyName string,
) (*armrecoveryservicesbackup.AzureIaaSVMProtectionPolicy, error) {
	client, err := armrecoveryservicesbackup.NewProtectionPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create protection policies client: %w", err)
	}

	resp, err := client.Get(ctx, vaultName, vaultRG, policyName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get policy %q from vault %q: %w", policyName, vaultName, err)
	}

	policy, ok := resp.Properties.(*armrecoveryservicesbackup.AzureIaaSVMProtectionPolicy)
	if !ok {
		return nil, fmt.Errorf("policy %q is not an AzureIaaSVMProtectionPolicy (got %T)", policyName, resp.Properties)
	}
	return policy, nil
}

// CreateOrUpdateBackupPolicyVM upserts the backup policy definition in the vault
// using the spec from the resolved VM.
func (c *Client) CreateOrUpdateBackupPolicyVM(ctx context.Context, vm azure.ResolvedVM) error {
	client, err := armrecoveryservicesbackup.NewProtectionPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create protection policies client: %w", err)
	}

	policy := buildVMProtectionPolicy(vm)

	_, err = client.CreateOrUpdate(ctx, vm.VaultName, vm.Resource.Vault.ResourceGroup, vm.PolicyName, policy, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert policy %q in vault %q: %w", vm.PolicyName, vm.VaultName, err)
	}
	return nil
}

func buildVMProtectionPolicy(vm azure.ResolvedVM) armrecoveryservicesbackup.ProtectionPolicyResource {
	spec := vm.Spec

	var schedPolicy armrecoveryservicesbackup.SchedulePolicyClassification
	var anchor time.Time
	var policyType armrecoveryservicesbackup.IAASVMPolicyType

	switch spec.PolicyType {
	case "Enhanced":
		policyType = armrecoveryservicesbackup.IAASVMPolicyTypeV2
		schedPolicy, anchor = buildVMSchedulePolicyV2(spec.Schedule.Full)
	case "Standard":
		policyType = armrecoveryservicesbackup.IAASVMPolicyTypeV1
		schedPolicy, anchor = buildVMSchedulePolicyV1(spec.Schedule.Full)
	default:
		// Throw error? This is for validation I suppose
	}

	retPolicy := buildVMLongTermRetentionPolicy(spec.Retention, anchor)

	iaasvmPolicy := &armrecoveryservicesbackup.AzureIaaSVMProtectionPolicy{
		TimeZone:                      to.Ptr(vm.Timezone),
		InstantRpRetentionRangeInDays: to.Ptr(int32(spec.InstantRPRetentionDays)),
		SchedulePolicy:                schedPolicy,
		RetentionPolicy:               retPolicy,
		PolicyType:                    to.Ptr(policyType),
	}

	if spec.InstantRPResourceGroupPrefix != "" || spec.InstantRPResourceGroupSuffix != "" {
		iaasvmPolicy.InstantRPDetails = &armrecoveryservicesbackup.InstantRPAdditionalDetails{
			AzureBackupRGNamePrefix: to.Ptr(spec.InstantRPResourceGroupPrefix),
			AzureBackupRGNameSuffix: to.Ptr(spec.InstantRPResourceGroupSuffix),
		}
	}

	return armrecoveryservicesbackup.ProtectionPolicyResource{
		Properties: iaasvmPolicy,
	}
}

// buildVMSchedulePolicyV1 converts the local FullSchedule into a V1 SimpleSchedulePolicy.
func buildVMSchedulePolicyV1(sched *policies.FullSchedule) (armrecoveryservicesbackup.SchedulePolicyClassification, time.Time) {
	anchor := time.Date(2016, 1, 1, sched.ParsedTime.Hour(), sched.ParsedTime.Minute(), 0, 0, time.UTC)

	switch sched.Frequency {
	case "weekly":
		day := armrecoveryservicesbackup.DayOfWeek(sched.Day)
		p := &armrecoveryservicesbackup.SimpleSchedulePolicy{
			ScheduleRunTimes:     []*time.Time{to.Ptr(anchor)},
			ScheduleRunFrequency: to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeWeekly),
			ScheduleRunDays:      []*armrecoveryservicesbackup.DayOfWeek{&day},
		}
		return p, anchor
	default:
		p := &armrecoveryservicesbackup.SimpleSchedulePolicy{
			ScheduleRunTimes:     []*time.Time{to.Ptr(anchor)},
			ScheduleRunFrequency: to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeDaily),
		}
		return p, anchor
	}
}

// buildVMSchedulePolicyV2 converts the local FullSchedule into a V2 (Enhanced) SimpleSchedulePolicyV2.
func buildVMSchedulePolicyV2(sched *policies.FullSchedule) (armrecoveryservicesbackup.SchedulePolicyClassification, time.Time) {
	anchor := time.Date(2016, 1, 1, sched.ParsedTime.Hour(), sched.ParsedTime.Minute(), 0, 0, time.UTC)

	switch sched.Frequency {
	case "hourly":
		p := &armrecoveryservicesbackup.SimpleSchedulePolicyV2{
			ScheduleRunFrequency: to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeHourly),
			HourlySchedule: &armrecoveryservicesbackup.HourlySchedule{
				Interval:                to.Ptr(int32(sched.Schedule)),
				ScheduleWindowStartTime: to.Ptr(anchor),
				ScheduleWindowDuration:  to.Ptr(int32(sched.Duration)),
			},
		}
		return p, anchor
	case "weekly":
		day := armrecoveryservicesbackup.DayOfWeek(sched.Day)
		p := &armrecoveryservicesbackup.SimpleSchedulePolicyV2{
			ScheduleRunFrequency: to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeWeekly),
			WeeklySchedule: &armrecoveryservicesbackup.WeeklySchedule{
				ScheduleRunDays:  []*armrecoveryservicesbackup.DayOfWeek{&day},
				ScheduleRunTimes: []*time.Time{to.Ptr(anchor)},
			},
		}
		return p, anchor
	default:
		p := &armrecoveryservicesbackup.SimpleSchedulePolicyV2{
			ScheduleRunFrequency: to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeDaily),
			DailySchedule: &armrecoveryservicesbackup.DailySchedule{
				ScheduleRunTimes: []*time.Time{to.Ptr(anchor)},
			},
		}
		return p, anchor
	}
}

// buildVMLongTermRetentionPolicy converts the local AzureRetention into an Azure LongTermRetentionPolicy.
func buildVMLongTermRetentionPolicy(ret policies.AzureRetention, anchor time.Time) *armrecoveryservicesbackup.LongTermRetentionPolicy {
	retentionTimes := []*time.Time{to.Ptr(anchor)}
	p := &armrecoveryservicesbackup.LongTermRetentionPolicy{}

	if ret.Daily != nil {
		p.DailySchedule = &armrecoveryservicesbackup.DailyRetentionSchedule{
			RetentionTimes: retentionTimes,
			RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
				Count:        to.Ptr(int32(ret.Daily.Count)),
				DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeDays),
			},
		}
	}

	if ret.Weekly != nil {
		day := armrecoveryservicesbackup.DayOfWeek(ret.Weekly.Day)
		p.WeeklySchedule = &armrecoveryservicesbackup.WeeklyRetentionSchedule{
			RetentionTimes: retentionTimes,
			DaysOfTheWeek:  []*armrecoveryservicesbackup.DayOfWeek{&day},
			RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
				Count:        to.Ptr(int32(ret.Weekly.Count)),
				DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeWeeks),
			},
		}
	} else {
		p.WeeklySchedule = &armrecoveryservicesbackup.WeeklyRetentionSchedule{
			RetentionTimes: retentionTimes,
			DaysOfTheWeek:  []*armrecoveryservicesbackup.DayOfWeek{to.Ptr(armrecoveryservicesbackup.DayOfWeekSunday)},
			RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
				Count:        to.Ptr(int32(1)),
				DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeWeeks),
			},
		}
	}

	if ret.Monthly != nil {
		month := &armrecoveryservicesbackup.MonthlyRetentionSchedule{
			RetentionTimes: retentionTimes,
			RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
				Count:        to.Ptr(int32(ret.Monthly.Count)),
				DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeMonths),
			},
			RetentionScheduleFormatType: to.Ptr(armrecoveryservicesbackup.RetentionScheduleFormatWeekly),
		}
		if ret.Monthly.Day != "" && ret.Monthly.Week != "" {
			weekOfMonth := armrecoveryservicesbackup.WeekOfMonth(ret.Monthly.Week)
			dayOfWeek := armrecoveryservicesbackup.DayOfWeek(ret.Monthly.Day)
			month.RetentionScheduleWeekly = &armrecoveryservicesbackup.WeeklyRetentionFormat{
				WeeksOfTheMonth: []*armrecoveryservicesbackup.WeekOfMonth{&weekOfMonth},
				DaysOfTheWeek:   []*armrecoveryservicesbackup.DayOfWeek{&dayOfWeek},
			}
		}
		p.MonthlySchedule = month
	} else {
		p.MonthlySchedule = nil
	}

	if ret.Yearly != nil {
		monthOfYear := armrecoveryservicesbackup.MonthOfYear(ret.Yearly.Month)
		weekOfMonth := armrecoveryservicesbackup.WeekOfMonth(ret.Yearly.Week)
		dayOfWeek := armrecoveryservicesbackup.DayOfWeek(ret.Yearly.Day)
		p.YearlySchedule = &armrecoveryservicesbackup.YearlyRetentionSchedule{
			RetentionTimes: retentionTimes,
			MonthsOfYear:   []*armrecoveryservicesbackup.MonthOfYear{&monthOfYear},
			RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
				Count:        to.Ptr(int32(ret.Yearly.Count)),
				DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeYears),
			},
			RetentionScheduleFormatType: to.Ptr(armrecoveryservicesbackup.RetentionScheduleFormatWeekly),
			RetentionScheduleWeekly: &armrecoveryservicesbackup.WeeklyRetentionFormat{
				WeeksOfTheMonth: []*armrecoveryservicesbackup.WeekOfMonth{&weekOfMonth},
				DaysOfTheWeek:   []*armrecoveryservicesbackup.DayOfWeek{&dayOfWeek},
			},
		}
	} else {
		p.YearlySchedule = nil
	}

	return p
}

// ApplyVM registers the VM as a protected item under the named policy.
func (c *Client) ApplyVM(ctx context.Context, rt azure.ResolvedVM) error {
	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create protected items client: %w", err)
	}

	rg := rt.ResourceGroup
	vmName := rt.ResourceName
	vaultName := rt.VaultName
	vaultRG := rt.Resource.Vault.ResourceGroup
	policyName := rt.PolicyName
	diskSelection := rt.DiskSelection

	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + rg + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + rg + ";" + vmName

	policyID, err := c.GetBackupPolicyID(ctx, vaultRG, vaultName, policyName)
	if err != nil {
		return fmt.Errorf("failed to look up policy ID for %q: %w", policyName, err)
	}
	sourceResourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/virtualMachines/%s",
		c.SubscriptionID, rg, vmName)

	var props armrecoveryservicesbackup.ProtectedItemResource
	switch {
	case diskSelection == nil || diskSelection.Mode == "all":
		props = buildVMProtectedItem(policyName, policyID, sourceResourceID, make([]*int32, 0), to.Ptr(false))
	case diskSelection.Mode == "include":
		props = buildVMProtectedItem(policyName, policyID, sourceResourceID, toLUNPtrs(diskSelection.LUNs), to.Ptr(true))
	case diskSelection.Mode == "exclude":
		props = buildVMProtectedItem(policyName, policyID, sourceResourceID, toLUNPtrs(diskSelection.LUNs), to.Ptr(false))
	default:
		return fmt.Errorf("invalid disk_selection.mode %q: expected all, include, or exclude", diskSelection.Mode)
	}
	_, err = client.CreateOrUpdate(ctx, vaultName, vaultRG, "Azure", containerName, protectedItemName, props, nil)
	if err != nil {
		return fmt.Errorf("failed to apply backup policy for VM %q: %w", vmName, err)
	}
	return nil
}

func buildVMProtectedItem(policyName, policyID, sourceResourceID string, lunList []*int32, isInclusionList *bool) armrecoveryservicesbackup.ProtectedItemResource {
	return armrecoveryservicesbackup.ProtectedItemResource{
		Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
			ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
				DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
					DiskLunList:     lunList,
					IsInclusionList: isInclusionList,
				},
			},
			PolicyName:       to.Ptr(policyName),
			PolicyID:         to.Ptr(policyID),
			SourceResourceID: to.Ptr(sourceResourceID),
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
