package assignments

type VMAssignment struct {
	Policy  string     `yaml:"policy"  validate:"required"`
	Targets []VMTarget `yaml:"targets" validate:"required"`
}

type VMTarget struct {
	ResourceGroup string         `yaml:"resource_group" validate:"required"`
	ResourceName  string         `yaml:"resource_name"  validate:"required"`
	DiskSelection *DiskSelection `yaml:"disk_selection" validate:"omitempty"`
}

// DiskSelection controls which disks on a VM are included in the backup.
// Omitting disk_selection on a target backs up all disks.
// Mode "all"     — back up every disk; luns is ignored.
// Mode "include" — back up only the listed LUNs; isInclusionList=true in Azure.
// Mode "exclude" — back up all disks except the listed LUNs; isInclusionList=false.
type DiskSelection struct {
	Mode string  `yaml:"mode" validate:"required,oneof=all include exclude"`
	LUNs []int32 `yaml:"luns" validate:"omitempty,dive,min=0"`
}
