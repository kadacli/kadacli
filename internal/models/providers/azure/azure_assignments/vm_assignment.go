package azure_assignments

// AzureAssignment is the common Azure assignment block - Layer 2.
type VMConfig struct {
	Assignments []VMAssignment `yaml:"assignments"`
}

type VMAssignment struct {
	Policy  string     `yaml:"policy" validate:"required"`
	Targets []VMTarget `yaml:"targets" validate:"required"`
}

type VMTarget struct {
	ResourceGroup      string `yaml:"resource_group validate: required"`
	ResourceName       string `yaml:"resource_name validate: required"`
	IncludeFutureDisks bool   `yaml:"include_future_disks validate: required,oneof=true false"`
}
