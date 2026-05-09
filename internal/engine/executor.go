package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

func Authenticate() (*azidentity.DefaultAzureCredential, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain credentials: %w", err)
	}
	return cred, nil
}

func GetAzureVault(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID, resourceGroup, vaultName string,
) (*armrecoveryservices.Vault, error) {
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

func GetAzureBackupPolicy(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID, resourceGroup, vaultName, policyName string,
) (string, error) {
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

// GetBackupPolicyVM retrieves the current backup configuration for an Azure VM.
func GetBackupPolicyVM(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID, resourceGroup, vaultName, vmName string,
) (armrecoveryservicesbackup.ProtectedItemsClientGetResponse, error) {
	var result armrecoveryservicesbackup.ProtectedItemsClientGetResponse

	containerName := "iaasvmcontainer;iaasvmcontainerv2;" + resourceGroup + ";" + vmName
	protectedItemName := "vm;iaasvmcontainerv2;" + resourceGroup + ";" + vmName

	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(subscriptionID, cred, nil)
	if err != nil {
		return result, fmt.Errorf("failed to create protected items client: %w", err)
	}

	result, err = client.Get(
		context.Background(),
		vaultName,
		resourceGroup,
		"Azure",
		containerName,
		protectedItemName,
		nil,
	)
	if err != nil {
		return result, fmt.Errorf("failed to get backup policy for VM %q in vault %q: %w", vmName, vaultName, err)
	}
	return result, nil
}

// GetBackupPolicyDB retrieves the current workload backup policy for a SQL target.
func GetBackupPolicyDB(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID, resourceGroup, vaultName, policyName string,
) (*armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy, error) {
	client, err := armrecoveryservicesbackup.NewProtectionPoliciesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create protection policies client: %w", err)
	}

	resp, err := client.Get(context.Background(), vaultName, resourceGroup, policyName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup policy %q from vault %q: %w", policyName, vaultName, err)
	}

	sqlPolicy, ok := resp.Properties.(*armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy)
	if !ok {
		return nil, fmt.Errorf("policy %q is not an AzureVMWorkloadProtectionPolicy (got %T)", policyName, resp.Properties)
	}
	return sqlPolicy, nil
}

// ApplyVM applies the backup policy from a ResolvedTarget to an Azure VM.
func ApplyVM(
	cred *azidentity.DefaultAzureCredential,
	subscriptionID string,
	rt models.ResolvedTarget,
) error {
	client, err := armrecoveryservicesbackup.NewProtectedItemsClient(subscriptionID, cred, nil)
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

	var properties armrecoveryservicesbackup.ProtectedItemResource

	switch {
	case len(diskLUNs) == 0 && includeFuture:
		properties = armrecoveryservicesbackup.ProtectedItemResource{
			Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
				ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
					DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
						DiskLunList:     make([]*int32, 0),
						IsInclusionList: to.Ptr(false),
					},
				},
				PolicyName: to.Ptr(policyName),
			},
		}
	case len(diskLUNs) > 0 && includeFuture:
		properties = armrecoveryservicesbackup.ProtectedItemResource{
			Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
				ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
					DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
						DiskLunList:     toLUNPointers(diskLUNs),
						IsInclusionList: to.Ptr(false),
					},
				},
				PolicyName: to.Ptr(policyName),
			},
		}
	case len(diskLUNs) > 0 && !includeFuture:
		properties = armrecoveryservicesbackup.ProtectedItemResource{
			Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
				ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
					DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
						DiskLunList:     toLUNPointers(diskLUNs),
						IsInclusionList: to.Ptr(true),
					},
				},
				PolicyName: to.Ptr(policyName),
			},
		}
	default:
		return fmt.Errorf("invalid disk selection configuration: no disks and includeFutureDisks=false")
	}

	_, err = client.CreateOrUpdate(
		context.Background(),
		vaultName,
		vaultRG,
		"Azure",
		containerName,
		protectedItemName,
		properties,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to apply backup policy for VM %q: %w", vmName, err)
	}
	return nil
}

func toLUNPointers(luns []int32) []*int32 {
	ptrs := make([]*int32, len(luns))
	for i, lun := range luns {
		ptrs[i] = to.Ptr(lun)
	}
	return ptrs
}

func CreateOrUpdateAzureBackupPolicy(
	ctx context.Context,
	client *armrecoveryservicesbackup.ProtectionPoliciesClient,
	cfg models.AzureRecoveryVMPolicy,
) (string, error) {
	policy := armrecoveryservicesbackup.ProtectionPolicyResource{
		Properties: &armrecoveryservicesbackup.AzureIaaSVMProtectionPolicy{
			BackupManagementType:          to.Ptr("AzureIaasVM"),
			TimeZone:                      to.Ptr("UTC"),
			InstantRpRetentionRangeInDays: to.Ptr(cfg.InstantRPRetentionDays),
			InstantRPDetails: &armrecoveryservicesbackup.InstantRPAdditionalDetails{
				AzureBackupRGNamePrefix: to.Ptr(cfg.InstantRPResourceGroupPrefix),
				AzureBackupRGNameSuffix: to.Ptr(cfg.InstantRPResourceGroupSuffix),
			},
			SchedulePolicy: &armrecoveryservicesbackup.SimpleSchedulePolicy{
				SchedulePolicyType:   to.Ptr("SimpleSchedulePolicy"),
				ScheduleRunFrequency: to.Ptr(armrecoveryservicesbackup.ScheduleRunTypeDaily),
				ScheduleRunTimes:     []*time.Time{to.Ptr(cfg.BackupTime)},
			},
			RetentionPolicy: &armrecoveryservicesbackup.LongTermRetentionPolicy{
				RetentionPolicyType: to.Ptr("LongTermRetentionPolicy"),
				DailySchedule: &armrecoveryservicesbackup.DailyRetentionSchedule{
					RetentionTimes: []*time.Time{to.Ptr(cfg.BackupTime)},
					RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
						Count:        to.Ptr(cfg.Retention.DailyCount),
						DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeDays),
					},
				},
				WeeklySchedule: &armrecoveryservicesbackup.WeeklyRetentionSchedule{
					DaysOfTheWeek:  []*armrecoveryservicesbackup.DayOfWeek{to.Ptr(cfg.Retention.WeeklyDay)},
					RetentionTimes: []*time.Time{to.Ptr(cfg.BackupTime)},
					RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
						Count:        to.Ptr(cfg.Retention.WeeklyCount),
						DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeWeeks),
					},
				},
				MonthlySchedule: &armrecoveryservicesbackup.MonthlyRetentionSchedule{
					RetentionScheduleFormatType: to.Ptr(armrecoveryservicesbackup.RetentionScheduleFormatWeekly),
					RetentionScheduleWeekly: &armrecoveryservicesbackup.WeeklyRetentionFormat{
						DaysOfTheWeek: []*armrecoveryservicesbackup.DayOfWeek{to.Ptr(cfg.Retention.WeeklyDay)},
						WeeksOfTheMonth: []*armrecoveryservicesbackup.WeekOfMonth{
							to.Ptr(armrecoveryservicesbackup.WeekOfMonthFirst),
						},
					},
					RetentionTimes: []*time.Time{to.Ptr(cfg.BackupTime)},
					RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
						Count:        to.Ptr(cfg.Retention.MonthlyCount),
						DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeMonths),
					},
				},
				YearlySchedule: &armrecoveryservicesbackup.YearlyRetentionSchedule{
					RetentionScheduleFormatType: to.Ptr(armrecoveryservicesbackup.RetentionScheduleFormatWeekly),
					MonthsOfYear: []*armrecoveryservicesbackup.MonthOfYear{
						to.Ptr(cfg.Retention.YearlyMonth),
					},
					RetentionScheduleWeekly: &armrecoveryservicesbackup.WeeklyRetentionFormat{
						DaysOfTheWeek: []*armrecoveryservicesbackup.DayOfWeek{
							to.Ptr(cfg.Retention.YearlyDay),
						},
						WeeksOfTheMonth: []*armrecoveryservicesbackup.WeekOfMonth{
							to.Ptr(cfg.Retention.YearlyWeek),
						},
					},
					RetentionTimes: []*time.Time{to.Ptr(cfg.BackupTime)},
					RetentionDuration: &armrecoveryservicesbackup.RetentionDuration{
						Count:        to.Ptr(cfg.Retention.YearlyCount),
						DurationType: to.Ptr(armrecoveryservicesbackup.RetentionDurationTypeYears),
					},
				},
			},
		},
	}

	resp, err := client.CreateOrUpdate(ctx, cfg.VaultName, cfg.ResourceGroup, cfg.PolicyName, policy, nil)
	if err != nil {
		return "", fmt.Errorf("creating/updating backup policy %q: %w", cfg.PolicyName, err)
	}

	if resp.ID == nil {
		return "", fmt.Errorf("backup policy %q created but response contained no ID", cfg.PolicyName)
	}

	return *resp.ID, nil
}
