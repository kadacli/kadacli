package models

// models package defines the structs used to define the backup policies

// The overall struct. Represents the entire configuration file
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

type Spec struct {
	Targets  []Target `mapstructure:"targets"`
	Timezone string   `mapstructure:"timezone"`
	Schedule Schedule `mapstructure:"schedule"`

	Retention struct {
		Daily   int `mapstructure:"daily"`
		Weekly  int `mapstructure:"weekly"`
		Monthly int `mapstructure:"monthly"`
	} `mapstructure:"retention"`

	Storage []struct {
		Primary struct {
			Provider   string `mapstructure:"provider"`
			Vault      string `mapstructure:"vault"`
			Redundancy string `mapstructure:"redundancy"`
		} `mapstructure:"primary"`
		Secondary struct {
			Provider   string `mapstructure:"provider"`
			Vault      string `mapstructure:"vault"`
			Redundancy string `mapstructure:"redundancy"`
		} `mapstructure:"secondary"`
	} `mapstructure:"storage"`
}
