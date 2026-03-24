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
	//Spec       Spec     `mapstructure:"spec"`
}

// The Metadata struct. Represents the metadata
type Metadata struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Owner       string `mapstructure:"owner"`
}

// The Spec Struct. Represents the specifics of the backup policy
// type Spec struct {
// 	//targets: array: selectors
// 	schedule: struct
// 	retention: struct
// 	storage struct
// }

func Parse() {
	v := load()

	var config Config
	var metadata Metadata

	err := v.Unmarshal(&config)
	if err != nil {
		panic(fmt.Errorf("unable to decode into struct, %w", err))
	}

	metadata = config.Metadata

	fmt.Println(config)
	fmt.Println(metadata)
}
