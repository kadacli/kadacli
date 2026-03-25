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

// The Spec Struct. Represents the specifics of the backup policy
// Consists of:
// a slice of Targets
// The backup schedule
// The retention policy
// The storage location
type Spec struct {
	// Targets is a slice of "Target" structs
	Targets []Target `mapstructure:"targets"`
	// The schedule has a type (full/incremental) and is defined as a cron
	schedule struct {
		Type struct {
			Cron string `mapstructure:"cron"`
		}
	}

	Timezone string

	// The retention policy.
	Retention struct {
		Daily   int `mapstructure:"daily"`
		Weekly  int `mapstructure:"weekly"`
		Monthly int `mapstructure:"monthly"`
	}

	// storage struct
	// A slice of priorities
	Storage []struct {
		// A priority determines where the target will be backup up
		Priority struct {
			Provider   string `mapstructure:"provider"`
			Vault      string `mapstructure:"vault"`
			Redundancy string `mapstructure:"redundancy"`
		}
	}
}
