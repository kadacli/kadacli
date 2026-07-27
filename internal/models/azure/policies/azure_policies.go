package policies

import "time"

// Retention and schedule types shared across all Azure resource kinds.

// VM
type AzureRetention struct {
	Daily   *DailyRetention   `yaml:"daily"   validate:"omitempty"`
	Weekly  *WeeklyRetention  `yaml:"weekly"  validate:"omitempty"`
	Monthly *MonthlyRetention `yaml:"monthly" validate:"omitempty"`
	Yearly  *YearlyRetention  `yaml:"yearly"  validate:"omitempty"`
}

// VM
type DailyRetention struct {
	Count int `yaml:"count" validate:"omitempty,min=1,max=9999"`
}

// VM
type WeeklyRetention struct {
	Day   string `yaml:"day"   validate:"omitempty,oneof=Monday Tuesday Wednesday Thursday Friday Saturday Sunday"`
	Count int    `yaml:"count" validate:"omitempty,min=1,max=5163"`
}

// VM
type MonthlyRetention struct {
	Day   string `yaml:"day"   validate:"omitempty,oneof=Monday Tuesday Wednesday Thursday Friday Saturday Sunday"`
	Week  string `yaml:"week"  validate:"omitempty,oneof=First Second Third Fourth Last"`
	Count int    `yaml:"count" validate:"required,min=1,max=1188"`
}

// VM
type YearlyRetention struct {
	Month string `yaml:"month" validate:"required,oneof=January February March April May June July August September October November December"`
	Week  string `yaml:"week"  validate:"required,oneof=First Second Third Fourth Last"`
	Day   string `yaml:"day"   validate:"required,oneof=Monday Tuesday Wednesday Thursday Friday Saturday Sunday"`
	Count int    `yaml:"count" validate:"required,min=1,max=99"`
}

// VM
type AzureSchedule struct {
	Full        *FullSchedule        `yaml:"full"        validate:"required"`
	Incremental *IncrementalSchedule `yaml:"incremental" validate:"omitempty"`
}

// VM
type FullSchedule struct {
	Frequency  string    `yaml:"frequency"  validate:"required,oneof=hourly daily weekly"`
	Time       string    `yaml:"time"       validate:"required,datetime=15:04"`
	Day        string    `yaml:"day"        validate:"omitempty,oneof=Monday Tuesday Wednesday Thursday Friday Saturday Sunday"`
	ParsedTime time.Time `yaml:"-"` // populated by the resolver, not from YAML
	Schedule   int       `yaml:"schedule"        validate:"omitempty,oneof=4 6 8 12"`
	Duration   int       `yaml:"duration"        validate:"omitempty,oneof=4 8 12 16 20 24"`
}

// VM
type IncrementalSchedule struct {
	Frequency string `yaml:"frequency" validate:"required,eq=daily"`
	Time      string `yaml:"time"      validate:"required,datetime=15:04"`
}

// TODO: Move this to disk.go
type AzureDiskSchedule struct {
	Frequency string `yaml:"frequency"  validate:"required,oneof=hourly daily"`
	Interval  int    `yaml:"interval"   validate:"omitempty,oneof=1 2 4 6 8 12"` // Only required if frequency == hourly
	Time      string `yaml:"time"       validate:"omitempty,datetime=15:04"`     // Only requred if frequency == daily
}

type AzureDiskRetention struct {
	Default        bool                     `yaml:"default"` // If default the rest is optional
	RetentionRules []AzureDiskRetentionRule `yaml:"retention_rules" validate:"omitempty,dive"`
}

type AzureDiskRetentionRule struct {
	RetentionRule     string `yaml:"rule" validate:"required,oneof=daily weekly"`
	RetentionDuration int    `yaml:"duration" validate:"omitempty,min=1,max=436"`
}
