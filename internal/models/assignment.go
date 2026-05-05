package models

// Assignment maps Policy definitions to specific Azure resources. One file per subscription.
type Assignment struct {
	APIVersion string             `yaml:"apiVersion" mapstructure:"apiVersion"`
	Kind       string             `yaml:"kind"       mapstructure:"kind"`
	Metadata   AssignmentMetadata `yaml:"metadata"   mapstructure:"metadata"`
	Spec       AssignmentSpec     `yaml:"spec"       mapstructure:"spec"`
}

type AssignmentMetadata struct {
	Name           string `yaml:"name"           mapstructure:"name"`
	SubscriptionID string `yaml:"subscriptionId" mapstructure:"subscriptionId"`
	Environment    string `yaml:"environment"    mapstructure:"environment"`
	Owner          string `yaml:"owner"          mapstructure:"owner"`
}

type AssignmentSpec struct {
	Assignments []PolicyAssignment `yaml:"assignments" mapstructure:"assignments"`
}

type PolicyAssignment struct {
	Policy  string             `yaml:"policy"  mapstructure:"policy"`
	Targets []AssignmentTarget `yaml:"targets" mapstructure:"targets"`
}

type AssignmentTarget struct {
	Type               string           `yaml:"type"                mapstructure:"type"`
	ResourceGroup      string           `yaml:"resource_group"      mapstructure:"resource_group"`
	ResourceName       string           `yaml:"resource_name"       mapstructure:"resource_name"`
	DiskSelection      []int32          `yaml:"disk_selection,omitempty"      mapstructure:"disk_selection"`
	IncludeFutureDisks bool             `yaml:"include_future_disks,omitempty" mapstructure:"include_future_disks"`
	Overrides          *PolicyOverrides `yaml:"overrides,omitempty" mapstructure:"overrides"`
}

type PolicyOverrides struct {
	Retention *RetentionSpec `yaml:"retention,omitempty" mapstructure:"retention"`
	Schedule  *ScheduleSpec  `yaml:"schedule,omitempty"  mapstructure:"schedule"`
}

// Resource represents a discovered live Azure resource.
type Resource struct {
	Type           string
	Name           string
	ResourceGroup  string
	SubscriptionID string
	Provider       string
	Tags           map[string]string
}

// ResolvedTarget pairs an AssignmentTarget with its fully resolved Policy (overrides applied).
type ResolvedTarget struct {
	SubscriptionID string
	PolicyName     string
	Target         AssignmentTarget
	Policy         Policy
}
