// internal/models/policy.go
package models

import "go.yaml.in/yaml/v3"

// BackupPolicy is the top-level, provider-agnostic envelope.
// The Resource and Spec are provider-specific and resolved
// based on the Provider field.
type Policy struct {
	APIVersion string   `yaml:"apiVersion" validate:"required,eq=backupctl/v1"`
	Kind       string   `yaml:"kind"       validate:"required,eq=Policy"`
	Provider   string   `yaml:"provider"   validate:"required,oneof=azure aws gcp veeam commvault"`
	Metadata   Metadata `yaml:"metadata"   validate:"required"`

	// Resource and Spec are kept as raw YAML until we know the provider,
	// then decoded into the correct provider-specific struct.
	Resource yaml.Node `yaml:"resource"`
	Spec     yaml.Node `yaml:"spec"`
}

type Metadata struct {
	Name        string `yaml:"name"        validate:"required"`
	Environment string `yaml:"environment" validate:"required,oneof=production staging development"`
	Timezone    string `yaml:"timezone"    validate:"required"`
	Owner       string `yaml:"owner"       validate:"required,email"`
}

type PathPolicy struct {
	Path   string `yaml:"path"`
	Policy Policy `yaml:"policy"`
}
