package models

// Assignment is the top-level document for an assignment YAML file.
type Assignment struct {
	APIVersion string             `yaml:"apiVersion"`
	Kind       string             `yaml:"kind"`
	Metadata   AssignmentMetadata `yaml:"metadata"`
	Spec       AssignmentSpec     `yaml:"spec"`
}

// AssignmentMetadata holds identifying information about the assignment.
type AssignmentMetadata struct {
	Name           string `yaml:"name"`
	SubscriptionID string `yaml:"subscriptionId"`
	Environment    string `yaml:"environment"`
	Owner          string `yaml:"owner"`
}

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
