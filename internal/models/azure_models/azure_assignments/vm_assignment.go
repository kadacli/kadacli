// internal/models/azure_models/azure_assignments/vm_assignment.go
package azure_assignements

type VMConfig struct {
	Assignments []VMAssignment `yaml:"assignments"`
}

type VMAssignment struct {
	Policy  string     `yaml:"policy"  validate:"required"`
	Targets []VMTarget `yaml:"targets" validate:"required"`
}

type VMTarget struct {
	ResourceGroup      string `yaml:"resource_group"       validate:"required"`
	ResourceName       string `yaml:"resource_name"        validate:"required"`
	IncludeFutureDisks bool   `yaml:"include_future_disks"`
}
