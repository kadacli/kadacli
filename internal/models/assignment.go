// internal/models/assignment.go
package models

import "go.yaml.in/yaml/v3"

type Assignment struct {
	APIVersion  string             `yaml:"apiVersion" validate:"required,eq=backupctl/v1"`
	Kind        string             `yaml:"kind"       validate:"required,eq=Assignment"`
	Provider    string             `yaml:"provider"   validate:"required,oneof=azure aws gcp veeam commvault"`
	Metadata    AssignmentMetadata `yaml:"metadata"   validate:"required"`
	Assignments yaml.Node          `yaml:"assignments"`
}

type AssignmentMetadata struct {
	Name        string `yaml:"name"        validate:"required"`
	Environment string `yaml:"environment" validate:"required,oneof=production staging development"`
	Owner       string `yaml:"owner"       validate:"required,email"`
}

type PathAssignment struct {
	Path       string     `yaml:"path"`
	Assignment Assignment `yaml:"assignment"`
}
