package engine

// The parser part of the engine.
// Receives a viper struct, unmarshals the information into structs and returns them.

import (
	"fmt"
)

// The overall struct. Represents the entire configuration file
type Config struct {
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
	selectors map[string]string `mapstructure:"selectors"`
}

// The Spec Struct. Represents the specifics of the backup policy
// Consists of:
// a slice of Targets
// The backup schedule
// The retention policy
// The storage location
type Spec struct {
	Targets []Target `mapstructure:"targets"`
	// The schedule has a type (full/incremental) and is defined as a cron
	schedule struct {
		Type struct {
			cron string
		}
	}

	Timezone string

	// The retention policy.
	Retention struct {
		Daily   int
		Weekly  int
		Monthly int
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

func Parse() {
	v := load()

	var config Config
	var metadata Metadata
	var spec Spec
	var targets []Target

	// Unmarshal the entire config file into a single config struct
	err := v.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("unable to decode into struct, %w", err))
	}

	// Get the data from the config struct and save them to the individual structs
	metadata = config.Metadata
	spec = config.Spec
	targets = spec.Targets

	fmt.Printf("metadata: %s \n", metadata)

	for i, target := range targets {
		fmt.Println(i, target.selectors["resource_group"])
	}

}
