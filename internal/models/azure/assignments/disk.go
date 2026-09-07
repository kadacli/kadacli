package assignments

type DiskAssignment struct {
	Policy  string       `yaml:"policy"`
	Targets []DiskTarget `yaml:"targets"`
}

type DiskTarget struct {
	ResourceGroup string     `yaml:"resource_group" validate:"required"`
	ResourceName  string     `yaml:"resource_name"  validate:"required"`
	Overrides     *Overrides `yaml:"overrides,omitempty"`
}

type Overrides struct {
	Retention *Retention `yaml:"retention,omitempty"`
}

type Retention struct {
	Operational *Operational `yaml:"operational,omitempty"`
}

type Operational struct {
	Count int `yaml:"count"`
}
