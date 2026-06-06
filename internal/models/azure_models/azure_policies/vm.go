package azure_policies

// AzureVMSpec is the VM-specific policy spec — Layer 3.
// Uses the shared retention and schedule types from azure_policies.go.
type AzureVMSpec struct {
	InstantRPRetentionDays       int    `yaml:"instant_rp_retention_days"        validate:"required,min=1,max=5"`
	InstantRPResourceGroupPrefix string `yaml:"instant_rp_resource_group_prefix" validate:"omitempty"`
	InstantRPResourceGroupSuffix string `yaml:"instant_rp_resource_group_suffix" validate:"omitempty"`

	Retention     AzureRetention `yaml:"retention"      validate:"required"`
	Schedule      AzureSchedule  `yaml:"schedule"       validate:"required"`
	DiskSelection *DiskSelection `yaml:"disk_selection" validate:"omitempty"`
}

// DiskSelection controls which disks are included in the backup.
// Mode "all" ignores the LUN list entirely.
// Mode "include" backs up only the listed LUNs.
// Mode "exclude" backs up all disks except the listed LUNs.
type DiskSelection struct {
	Mode string  `yaml:"mode" validate:"required,oneof=all include exclude"`
	LUNs []int32 `yaml:"luns" validate:"omitempty,dive,min=0"`
}
