package models

// Defines the fields in the backupPolicy that is pulled from Azure
type AzureVMBackupPolicy struct {
	PolicyName         string `json:"policyName"`
	PolicyId           string `json:"policyId"`
	ExtendedProperties struct {
		DiskExclusionProperties struct {
			DiskLunList     []int `json:"diskLunList"`
			IsInclusionList bool  `json:"isInclusionList"`
		} `json:"diskExclusionProperties"`
	} `json:"extendedProperties"`
}
