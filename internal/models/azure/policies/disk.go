package policies

// contains the "spec" of a disk policy in Azure

type AzureDiskSpec struct {
	SnapshotResourceGroup string             `yaml:"snapshot_resource_group"` // optional, defaults to disk's resource group
	Schedule              AzureDiskSchedule  `yaml:"schedule" validate:"required"`
	Retention             AzureDiskRetention `yaml:"retention" validate:"required"`
}
