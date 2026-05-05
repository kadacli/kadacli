package models

import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"

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
