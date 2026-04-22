package models

// models package defines the structs used to define the backup policies

// The overall struct. Represents the entire unvalidated configuration file
type RawPolicy struct {
	ApiVersion string   `mapstructure:"apiVersion"`
	Kind       string   `mapstructure:"kind"`
	Metadata   Metadata `mapstructure:"metadata"`
	Spec       Spec     `mapstructure:"spec"`
}

type Metadata struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Owner       string `mapstructure:"owner"`
}

type Target struct {
	Type      string          `mapstructure:"type"`
	Selectors []AzureSelector `mapstructure:"selectors"`
}

// Might extend to more selectors later
type AzureSelector struct {
	ResourceGroup string `mapstructure:"resource_group"`
	ResourceName  string `mapstructure:"resource_name"`
}
type StorageProvider struct {
	Provider           string  `mapstructure:"provider"`
	Vault              string  `mapstructure:"vault"`
	Resource_group     string  `mapstructure:"resource_group"` // Hard-coded to Azure
	PolicySubtype      string  `mapstructure:"policy_subtype"`
	BackupPolicy       string  `mapstructure:"backup_policy"`
	DiskSelection      []uint8 `mapstructure:"disk_selection"`
	IncludeFutureDisks bool    `mapstructure:"include_future_disks"`
}

type Spec struct {
	Targets         []Target        `mapstructure:"targets"`
	Timezone        string          `mapstructure:"timezone"`
	StorageProvider StorageProvider `mapstructure:"storage"`
}

// The BackupPolicy type
// Structureally identical to RawPolicy, but must only be created after validation
type BackupPolicy struct {
	ApiVersion string   `mapstructure:"apiVersion"`
	Kind       string   `mapstructure:"kind"`
	Metadata   Metadata `mapstructure:"metadata"`
	Spec       Spec     `mapstructure:"spec"`
}
