package models

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
)

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

type AzureSQLVMBackupPolicy struct {
	PolicyName string
	PolicyId   string
	// The SDK type for SQL VM workload policies
	Properties *armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy
}

type RetentionConfig struct {
	DailyCount   int32
	WeeklyCount  int32
	WeeklyDay    armrecoveryservicesbackup.DayOfWeek
	MonthlyCount int32
	YearlyCount  int32
}
