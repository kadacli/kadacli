package models

// models package defines the structs used to define the backup policies

// The overall struct. Represents the entire unvalidated configuration file
type RawPolicy struct {
	ApiVersion string   `mapstructure:"apiVersion"`
	Kind       string   `mapstructure:"kind"`
	Metadata   Metadata `mapstructure:"metadata"`
	Spec       Spec     `mapstructure:"spec"`
}

// The Metadata struct. Represents the metadata
type Metadata struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Owner       string `mapstructure:"owner"`
}

// The target struct. Defines the target that will be backed up.
// Consists of the type (of resource), defined as a string
// and the selector, which is a string:string map
type Target struct {
	Type      string            `mapstructure:"type"`
	Selectors map[string]string `mapstructure:"selectors"`
}

// The Schedule Struct. Defines the incremental or full backup schedule
// Type, can be "full" or "incremental"
type CronSchedule struct {
	Cron string `mapstructure:"cron"`
}

type Schedule struct {
	Full        CronSchedule `mapstructure:"full"`
	Incremental CronSchedule `mapstructure:"incremental"`
}

type Retention struct {
	Daily   int `mapstructure:"daily"`
	Weekly  int `mapstructure:"weekly"`
	Monthly int `mapstructure:"monthly"`
}

type StorageProvider struct {
	Provider   string `mapstructure:"provider"`
	Vault      string `mapstructure:"vault"`
	Redundancy string `mapstructure:"redundancy"`
}

type Storage struct {
	Primary   StorageProvider `mapstructure:"primary"`
	Secondary StorageProvider `mapstructure:"secondary"`
}

type Spec struct {
	Targets   []Target  `mapstructure:"targets"`
	Timezone  string    `mapstructure:"timezone"`
	Schedule  Schedule  `mapstructure:"schedule"`
	Retention Retention `mapstructure:"retention"`
	Storage   []Storage `mapstructure:"storage"`
}

// The BackupPolicy type
// Structureally identical to RawPolicy, but must only be created after validation
type BackupPolicy struct {
	ApiVersion string   `mapstructure:"apiVersion"`
	Kind       string   `mapstructure:"kind"`
	Metadata   Metadata `mapstructure:"metadata"`
	Spec       Spec     `mapstructure:"spec"`
}
