// AssignmentSpec holds the list of policy-to-target assignments.
type AssignmentSpec struct {
	Assignments []PolicyAssignment `yaml:"assignments"`
}

// PolicyAssignment maps a single policy to one or more backup targets.
type PolicyAssignment struct {
	Policy  string             `yaml:"policy"`
	Targets []AssignmentTarget `yaml:"targets"`
}

// AssignmentTarget identifies a single Azure resource to back up.
type AssignmentTarget struct {
	Type          ResourceType `yaml:"type"`
	ResourceGroup string       `yaml:"resource_group"`
	ResourceName  string       `yaml:"resource_name"`
}

// ResourceType is the set of supported Azure resource types.
type ResourceType string

const (
	ResourceTypeAzureVM    ResourceType = "azure_vm"
	ResourceTypeAzureSQL   ResourceType = "azure_sql"
	ResourceTypeAzureSQLVM ResourceType = "azure_sql_vm"
)