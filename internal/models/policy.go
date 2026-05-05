package models

// Policy defines a reusable backup policy. One file per policy.
type Policy struct {
	APIVersion string         `yaml:"apiVersion" mapstructure:"apiVersion"`
	Kind       string         `yaml:"kind"       mapstructure:"kind"`
	Metadata   PolicyMetadata `yaml:"metadata"   mapstructure:"metadata"`
	Spec       PolicySpec     `yaml:"spec"       mapstructure:"spec"`
}

type PolicyMetadata struct {
	Name        string `yaml:"name"        mapstructure:"name"`
	Environment string `yaml:"environment" mapstructure:"environment"`
	Owner       string `yaml:"owner"       mapstructure:"owner"`
}

type PolicySpec struct {
	Timezone  string        `yaml:"timezone"  mapstructure:"timezone"`
	Vault     VaultRef      `yaml:"vault"     mapstructure:"vault"`
	Retention RetentionSpec `yaml:"retention" mapstructure:"retention"`
	Schedule  ScheduleSpec  `yaml:"schedule"  mapstructure:"schedule"`
}

type VaultRef struct {
	Name          string `yaml:"name"           mapstructure:"name"`
	ResourceGroup string `yaml:"resource_group" mapstructure:"resource_group"`
}

type RetentionSpec struct {
	Daily   *DailyRetention   `yaml:"daily,omitempty"   mapstructure:"daily"`
	Weekly  *WeeklyRetention  `yaml:"weekly,omitempty"  mapstructure:"weekly"`
	Monthly *MonthlyRetention `yaml:"monthly,omitempty" mapstructure:"monthly"`
	Yearly  *YearlyRetention  `yaml:"yearly,omitempty"  mapstructure:"yearly"`
}

type DailyRetention struct {
	Count int `yaml:"count" mapstructure:"count"`
}

type WeeklyRetention struct {
	Day   string `yaml:"day"   mapstructure:"day"`
	Count int    `yaml:"count" mapstructure:"count"`
}

type MonthlyRetention struct {
	Count int `yaml:"count" mapstructure:"count"`
}

type YearlyRetention struct {
	Count int `yaml:"count" mapstructure:"count"`
}

type ScheduleSpec struct {
	Full         *FullSchedule         `yaml:"full,omitempty"         mapstructure:"full"`
	Differential *DifferentialSchedule `yaml:"differential,omitempty" mapstructure:"differential"`
	Log          *LogSchedule          `yaml:"log,omitempty"          mapstructure:"log"`
}

type FullSchedule struct {
	Frequency string `yaml:"frequency" mapstructure:"frequency"` // daily | weekly
	Time      string `yaml:"time"      mapstructure:"time"`
}

type DifferentialSchedule struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type LogSchedule struct {
	FrequencyMinutes int `yaml:"frequency_minutes" mapstructure:"frequency_minutes"`
}
