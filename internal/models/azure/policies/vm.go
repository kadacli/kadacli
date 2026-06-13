package policies

// AzureVMSpec is the VM-specific policy spec — Layer 3.
// Uses the shared retention and schedule types from azure_policies.go.
type AzureVMSpec struct {
	PolicyType                   string `yaml:"policy_type"  validate:"required,oneof=Standard Enhanced"`
	InstantRPRetentionDays       int    `yaml:"instant_rp_retention_days"        validate:"required,min=1,max=5"`
	InstantRPResourceGroupPrefix string `yaml:"instant_rp_resource_group_prefix" validate:"omitempty"`
	InstantRPResourceGroupSuffix string `yaml:"instant_rp_resource_group_suffix" validate:"omitempty"`

	Retention AzureRetention `yaml:"retention" validate:"required"`
	Schedule  AzureSchedule  `yaml:"schedule"  validate:"required"`
}
