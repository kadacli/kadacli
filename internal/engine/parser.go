package engine

// In internal/engine/parser.go implement ValidatePolicy(policy BackupPolicy, sourcePath string) error.
// Run go-playground/validator on the fully resolved struct.
// If validation fails wrap the error with the source file path so the message
//   reads: policies/production/finance.yaml: metadata.owner is required.
// Runs after resolution so it validates the merged result.

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

func Parse() {
	errors, policies := LoadAllPolicies("./example_yaml")

	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Printf("unable to load config file:  %w \n", err)
		}
	}

	if len(policies) > 0 {
		for i, policy := range policies {
			fmt.Println(i, policy)
		}
	}

	//var config Config

	// Unmarshal the entire config file into a single config struct
	// err := v.Unmarshal(&config)
	// if err != nil {
	// 	panic(fmt.Errorf("unable to decode into struct, %w", err))
	// }

	// Print the data for debugging
	//Print_Data(config)

}

// Prints the configuration
// Inputs: A Config struct
// Returns: nil
// Purpose: See the data
func Print_Data(config Config) {

	var metadata Metadata
	var spec Spec
	var targets []Target

	// Get the data from the config variable and save them to individual variables
	metadata = config.Metadata
	spec = config.Spec
	targets = spec.Targets

	// Print the data
	fmt.Printf("config: %s, %s\n", config.ApiVersion, config.Kind)
	fmt.Printf("metadata: %s \n", metadata)
	fmt.Println("spec:")

	if len(targets) > 0 {
		for i, target := range targets {
			fmt.Println(i, target.Type, target.Selectors)
		}
	} else {
		panic(fmt.Errorf("No targets defined"))
	}

}
