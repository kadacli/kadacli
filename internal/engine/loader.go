package engine

// LoadAllPolicies(dir string) using filepath.WalkDir to recursively
//  find all *.yaml files under dir, skipping any path containing _defaults/.
// For each file call raw YAML unmarshal and return a slice of RawPolicy structs
//   including both the parsed BackupPolicy and the original file path.
// Return a combined error listing all files that failed to parse.

// Assumptions: The Go module will live in the root of a project directory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// The overall struct. Represents the entire configuration file
type RawPolicy struct {
	ApiVersion string   `mapstructure:"apiVersion"`
	Kind       string   `mapstructure:"kind"`
	Metadata   Metadata `mapstructure:"metadata"`
	Spec       Spec     `mapstructure:"spec"`
}

// function LoadAllPolicies
// Inputs: the basic search directory
// Returns: A slice of errors and a slice of RawPolicies
func LoadAllPolicies(dir string) ([]error, []RawPolicy) {
	var errors []error
	var rawPolicies []RawPolicy

	// find all yaml files
	var paths = findAllPolicies(dir)

	// try to load all yaml files
	for _, path := range paths {
		err, policy := loadPolicy(path)

		if err != nil {
			errors = append(errors, err)
		} else {
			rawPolicies = append(rawPolicies, policy)
		}
	}

	// Returns a slice of errors and a slice of raw polices
	return errors, rawPolicies
}

// Find the paths to all *.yaml or *.yml files (case insensitive)
// Walk through the entire project dir to find the files
// Returns a list of paths
func findAllPolicies(dir string) []string {
	var paths []string
	var root = dir

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isYAML(path) {
			paths = append(paths, path)
		}
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "walk error: %v\n", err)
	}

	return paths
}

// Unmarshals a single policy into a RawPolicy Struct
// takes a path
// Returns an error (on failure, otherwise nil) and a RawPolicy struct (on success, otherwise nil)
func loadPolicy(path string) (error, RawPolicy) {
	var err error
	var policy RawPolicy

	v := viper.New()
	v.SetConfigFile(path)

	err = v.ReadInConfig()

	err2 := v.Unmarshal(&policy)
	if err2 != nil {
		panic(fmt.Errorf("Unmarshal error for %s Unable to decode into struct, %w", path, err2))
	}

	return err, policy
}

// check if file is a yaml file
// Input: string
// returns bool
func isYAML(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
