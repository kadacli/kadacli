package azure_external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure/policies"
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
	// debugging:
	b, _ := json.MarshalIndent(policy, "", "  ")
	fmt.Println(string(b))

	_, err = client.CreateOrUpdate(ctx, vm.VaultName, vm.Resource.Vault.ResourceGroup, vm.PolicyName, policy, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert policy %q in vault %q: %w", vm.PolicyName, vm.VaultName, err)
	}
	return nil
}

func buildVMProtectionPolicy(vm azure.ResolvedVM) armrecoveryservicesbackup.ProtectionPolicyResource {
	spec := vm.Spec
	schedPolicy, anchor := buildVMSchedulePolicy(spec.Schedule.Full)
	retPolicy := buildVMLongTermRetentionPolicy(spec.Retention, anchor)

	iaasvmPolicy := &armrecoveryservicesbackup.AzureIaaSVMProtectionPolicy{
		TimeZone:                      to.Ptr(vm.Timezone),
		InstantRpRetentionRangeInDays: to.Ptr(int32(spec.InstantRPRetentionDays)),
		SchedulePolicy:                schedPolicy,
		RetentionPolicy:               retPolicy,
	}

	if spec.PolicyType != "Standard" {
		iaasvmPolicy.PolicyType = to.Ptr(armrecoveryservicesbackup.IAASVMPolicyTypeV2)
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

// buildVMSchedulePolicy converts the local FullSchedule into an Azure SimpleSchedulePolicy.
// Azure expects ScheduleRunTimes with a concrete datetime; 2016-01-01 is the conventional anchor.
func buildVMSchedulePolicy(sched *policies.FullSchedule) (*armrecoveryservicesbackup.SimpleSchedulePolicy, time.Time) {
	anchor := time.Date(2016, 1, 1, sched.ParsedTime.Hour(), sched.ParsedTime.Minute(), 0, 0, time.UTC)

	p := &armrecoveryservicesbackup.SimpleSchedulePolicy{
		ScheduleRunTimes: []*time.Time{to.Ptr(anchor)},
	}

	if sched.Frequency == "weekly" {
		day := armrecoveryservicesbackup.DayOfWeek(sched.Day)
		p.ScheduleRunFrequency = to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeWeekly)
		p.ScheduleRunDays = []*armrecoveryservicesbackup.DayOfWeek{&day}
	} else {
		p.ScheduleRunFrequency = to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeDaily)
	}

	return p, anchor
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

	var props armrecoveryservicesbackup.ProtectedItemResource
	switch {
	case diskSelection == nil || diskSelection.Mode == "all":
		props = buildVMProtectedItem(policyName, make([]*int32, 0), to.Ptr(false))
	case diskSelection.Mode == "include":
		props = buildVMProtectedItem(policyName, toLUNPtrs(diskSelection.LUNs), to.Ptr(true))
	case diskSelection.Mode == "exclude":
		props = buildVMProtectedItem(policyName, toLUNPtrs(diskSelection.LUNs), to.Ptr(false))
	default:
		return fmt.Errorf("invalid disk_selection.mode %q: expected all, include, or exclude", diskSelection.Mode)
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
