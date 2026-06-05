// internal/models/assignment.go
package models

import "go.yaml.in/yaml/v3"

type Assignment struct {
	ApiVersion  string             `yaml:"apiVersion"`
	Kind        string             `yaml:"kind"`
	Provider    string             `yaml:"provider"`
	Metadata    AssignmentMetadata `yaml:"metadata"`
	Resource    yaml.Node          `yaml:"resource"`
	Assignments yaml.Node          `yaml:"assignments"`
}

type AssignmentMetadata struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Owner       string `yaml:"owner"`
}
