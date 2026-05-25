package azure

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dataprotection/armdataprotection"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
)

type AzureRecoveryVMPolicy struct {
	ResourceGroup string
	VaultName     string
	PolicyName    string
	BackupTime    time.Time
	Retention     RetentionConfig

	InstantRPRetentionDays       int32
	InstantRPResourceGroupPrefix string
	InstantRPResourceGroupSuffix string
}

type AzureRecoveryDiskPolicy struct {
	ResourceGroup               string
	VaultName                   string
	PolicyName                  string
	BackupIntervalHours         int32
	RetentionDays               int32
	SnapshotResourceGroupPrefix string
	SnapshotResourceGroupSuffix string
}

// Defines the fields in the backupPolicy that is pulled from Azure
type AzureVMBackupPolicy struct {
	PolicyName         string `json:"policyName"`
	PolicyId           string `json:"policyId"`
	ExtendedProperties struct {
		DiskExclusionProperties struct {
			DiskLunList     []*int32 `json:"diskLunList"`
			IsInclusionList bool     `json:"isInclusionList"`
		} `json:"diskExclusionProperties"`
	} `json:"extendedProperties"`
}

type AzureSQLVMBackupPolicy struct {
	PolicyName string
	PolicyId   string
	// The SDK type for SQL VM workload policies
	Properties *armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy
}

type AzureDiskBackupPolicy struct {
	PolicyName                  string
	PolicyID                    string
	ResourceGroup               string
	VaultName                   string
	SnapshotResourceGroupPrefix string
	SnapshotResourceGroupSuffix string
	Properties                  *armdataprotection.BackupPolicy
}

type AzureSqlDbBackupPolicy struct {
	PolicyName string
	PolicyId   string
}

// Maps to a retention schedule in the azure portal
type RetentionConfig struct {
	DailyCount   int32
	WeeklyCount  int32
	WeeklyDay    armrecoveryservicesbackup.DayOfWeek
	MonthlyCount int32
	YearlyCount  int32
	YearlyMonth  armrecoveryservicesbackup.MonthOfYear
	YearlyDay    armrecoveryservicesbackup.DayOfWeek
	YearlyWeek   armrecoveryservicesbackup.WeekOfMonth
}
