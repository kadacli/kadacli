package engine

// In internal/engine/parser.go implement ValidatePolicy(policy BackupPolicy, sourcePath string) error.
// Run go-playground/validator on the fully resolved struct.
// If validation fails wrap the error with the source file path so the message
//   reads: policies/production/finance.yaml: metadata.owner is required.
// Runs after resolution so it validates the merged result.

import (
	"fmt"
)



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
