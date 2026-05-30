package azure_assignments

// AzureAssignment is the common Azure assignment block - Layer 2.
type AzureAssignment struct {
	Policies []AzureVMAssignment `yaml:"assignments" validate:"required"`
}

type AzureVMAssignment struct {
	Policy  string          `yaml:"policy" validate:"required"`
	Targets []AzureVMTarget `yaml:"targets" validate:"required"`
}

type AzureVMTarget struct {
	Type               string `yaml:"type validate: required,oneof=vm disk sqlvm sqlpaas"`
	ResourceGroup      string `yaml:"resource_group validate: required"`
	ResourceName       string `yaml:"resource_name validate: required"`
	IncludeFutureDisks bool   `yaml:"include_future_disks validate: required,oneof=true false"`
}
