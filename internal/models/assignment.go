package models

// Assignment is the top-level document for an assignment YAML file.
type Assignment struct {
	APIVersion string             `yaml:"apiVersion"`
	Provider   string             `yaml:"provider"`
	Metadata   AssignmentMetadata `yaml:"metadata"`
}

// AssignmentMetadata holds identifying information about the assignment.
type AssignmentMetadata struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Timezone    string `yaml:"timezone"` // TODO: Go timezone package?
	Owner       string `yaml:"owner"`
}


