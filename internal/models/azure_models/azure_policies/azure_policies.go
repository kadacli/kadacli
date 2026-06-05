package azure_policies

import "time"

// Retention and schedule types shared across all Azure resource kinds.

type AzureRetention struct {
	Daily   *DailyRetention   `yaml:"daily"   validate:"required"`
	Weekly  *WeeklyRetention  `yaml:"weekly"  validate:"omitempty"`
	Monthly *MonthlyRetention `yaml:"monthly" validate:"omitempty"`
	Yearly  *YearlyRetention  `yaml:"yearly"  validate:"omitempty"`
}

type DailyRetention struct {
	Count int `yaml:"count" validate:"required,min=1,max=9999"`
}

type WeeklyRetention struct {
	Day   string `yaml:"day"   validate:"required,oneof=Monday Tuesday Wednesday Thursday Friday Saturday Sunday"`
	Count int    `yaml:"count" validate:"required,min=1,max=5163"`
}

type MonthlyRetention struct {
	Day   string `yaml:"day"   validate:"omitempty,oneof=Monday Tuesday Wednesday Thursday Friday Saturday Sunday"`
	Week  string `yaml:"week"  validate:"omitempty,oneof=First Second Third Fourth Last"`
	Count int    `yaml:"count" validate:"required,min=1,max=1188"`
}

type YearlyRetention struct {
	Month string `yaml:"month" validate:"required,oneof=January February March April May June July August September October November December"`
	Week  string `yaml:"week"  validate:"required,oneof=First Second Third Fourth Last"`
	Day   string `yaml:"day"   validate:"required,oneof=Monday Tuesday Wednesday Thursday Friday Saturday Sunday"`
	Count int    `yaml:"count" validate:"required,min=1,max=99"`
}

type AzureSchedule struct {
	Full        *FullSchedule        `yaml:"full"        validate:"required"`
	Incremental *IncrementalSchedule `yaml:"incremental" validate:"omitempty"`
}

type FullSchedule struct {
	Frequency  string    `yaml:"frequency"  validate:"required,oneof=daily weekly"`
	Time       string    `yaml:"time"       validate:"required,datetime=15:04"`
	Day        string    `yaml:"day"        validate:"omitempty,oneof=Monday Tuesday Wednesday Thursday Friday Saturday Sunday"`
	ParsedTime time.Time `yaml:"-"` // populated by the resolver, not from YAML
}

type IncrementalSchedule struct {
	Frequency string `yaml:"frequency" validate:"required,eq=daily"`
	Time      string `yaml:"time"      validate:"required,datetime=15:04"`
}
