package engine

// LoadAllPolicies(dir string) using filepath.WalkDir to recursively
//  find all *.yaml files under dir, skipping any path containing _defaults/.
// For each file call raw YAML unmarshal and return a slice of RawPolicy structs
//   including both the parsed BackupPolicy and the original file path.
// Return a combined error listing all files that failed to parse.

// Assumptions: The Go module will live in the root of a project directory

import (
	"fmt"

	"github.com/spf13/viper"
)

// The overall struct. Represents the entire configuration file
type RawPolicy struct {
	ApiVersion string   `mapstructure:"apiVersion"`
	Kind       string   `mapstructure:"kind"`
	Metadata   Metadata `mapstructure:"metadata"`
	Spec       Spec     `mapstructure:"spec"`
}

func LoadAllPolicies(dir string) ([]error, []RawPolicy) {
	var errors []error
	var rawPolicies []RawPolicy

	// find all yaml files

	// try to load all yaml files

	// Returns a slice of errors and a slice of raw polices
	return errors, rawPolicies
}

// Find the paths to all *.yaml or *.yml files (case insensitive)
// Walk through the entire project dir to find the files
// Returns a list of paths
func FindAllPolicies() []string {
	var paths []string

	return paths
}

// Loads a single policy
// takes a path
// Returns an error (on failure, otherwise nil) and a RawPolicy struct (on success, otherwise nil)
func LoadPolicy(path string) (error, RawPolicy) {
	var error error
	var policy RawPolicy

	return error, policy
}

func Load() *viper.Viper {
	v := viper.New()
	v.SetConfigName("example_azure")
	v.AddConfigPath("example_yaml")

	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	return v
}
