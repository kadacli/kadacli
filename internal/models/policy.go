package models

// Policy is the top-level document, mirroring the YAML structure.
type Policy struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   PolicyMetadata `yaml:"metadata"`
	Spec       PolicySpec     `yaml:"spec"`
}

// PolicyMetadata holds identifying information about the policy.
type PolicyMetadata struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Owner       string `yaml:"owner"`
}

// PolicySpec is the core configuration of the backup policy.
type PolicySpec struct {
	Timezone  string          `yaml:"timezone"`
	Vault     VaultConfig     `yaml:"vault"`
	Retention RetentionConfig `yaml:"retention"`
	Schedule  ScheduleConfig  `yaml:"schedule"`
}

// VaultConfig identifies the backup vault to use.
type VaultConfig struct {
	Name          string `yaml:"name"`
	ResourceGroup string `yaml:"resource_group"`
}

// RetentionConfig defines how long each tier of backup is kept.
type RetentionConfig struct {
	Daily   DailyRetention   `yaml:"daily"`
	Weekly  WeeklyRetention  `yaml:"weekly"`
	Monthly MonthlyRetention `yaml:"monthly"`
	Yearly  YearlyRetention  `yaml:"yearly"`
}

type DailyRetention struct {
	Count int `yaml:"count"`
}

type WeeklyRetention struct {
	Day   string `yaml:"day"` // e.g. "Sunday"
	Count int    `yaml:"count"`
}

type MonthlyRetention struct {
	Count int `yaml:"count"`
}

type YearlyRetention struct {
	Count int `yaml:"count"`
}

// ScheduleConfig defines when and how often backups run.
type ScheduleConfig struct {
	Full         FullSchedule         `yaml:"full"`
	Log          LogSchedule          `yaml:"log"`
	Differential DifferentialSchedule `yaml:"differential"`
}

// FullSchedule defines the primary backup cadence.
type FullSchedule struct {
	Frequency string `yaml:"frequency"` // e.g. "daily"
	Time      string `yaml:"time"`      // e.g. "02:00"
}

// LogSchedule applies to resource types that support log backups (e.g. SQL).
// FrequencyMinutes is silently ignored for unsupported resource types.
type LogSchedule struct {
	FrequencyMinutes int `yaml:"frequency_minutes"`
}

// DifferentialSchedule applies to resource types that support differential
// backups. Enabled is silently ignored for unsupported resource types.
type DifferentialSchedule struct {
	Enabled bool `yaml:"enabled"`
}
