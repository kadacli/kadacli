package models

import "time"

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
	Timezone                     string        `yaml:"timezone"                      mapstructure:"timezone"`
	Vault                        VaultRef      `yaml:"vault"                         mapstructure:"vault"`
	Retention                    RetentionSpec `yaml:"retention"                     mapstructure:"retention"`
	Schedule                     ScheduleSpec  `yaml:"schedule"                      mapstructure:"schedule"`
	InstantRPRetentionDays       int32         `yaml:"instant_rp_retention_days"     mapstructure:"instant_rp_retention_days"`
	InstantRPResourceGroupPrefix string        `yaml:"instant_rp_resource_group_prefix" mapstructure:"instant_rp_resource_group_prefix"`
	InstantRPResourceGroupSuffix string        `yaml:"instant_rp_resource_group_suffix" mapstructure:"instant_rp_resource_group_suffix"`
	SnapshotResourceGroupPrefix  string        `yaml:"snapshot_resource_group_prefix" mapstructure:"snapshot_resource_group_prefix"`
	SnapshotResourceGroupSuffix  string        `yaml:"snapshot_resource_group_suffix" mapstructure:"snapshot_resource_group_suffix"`
}

type VaultRef struct {
	Name          string `yaml:"name"           mapstructure:"name"`
	ResourceGroup string `yaml:"resource_group" mapstructure:"resource_group"`
}

type RetentionSpec struct {
	Daily       *DailyRetention       `yaml:"daily,omitempty"       mapstructure:"daily"`
	Weekly      *WeeklyRetention      `yaml:"weekly,omitempty"      mapstructure:"weekly"`
	Monthly     *MonthlyRetention     `yaml:"monthly,omitempty"     mapstructure:"monthly"`
	Yearly      *YearlyRetention      `yaml:"yearly,omitempty"      mapstructure:"yearly"`
	Operational *OperationalRetention `yaml:"operational,omitempty" mapstructure:"operational"`
}

type DailyRetention struct {
	Count int32 `yaml:"count" mapstructure:"count"`
}

type WeeklyRetention struct {
	Day   string `yaml:"day"   mapstructure:"day"`
	Count int32  `yaml:"count" mapstructure:"count"`
}

type MonthlyRetention struct {
	Count int32 `yaml:"count" mapstructure:"count"`
}

type YearlyRetention struct {
	Month string `yaml:"month" mapstructure:"month"`
	Day   string `yaml:"day"   mapstructure:"day"`
	Week  string `yaml:"week"  mapstructure:"week"`
	Count int32  `yaml:"count" mapstructure:"count"`
}

type ScheduleSpec struct {
	Full         *FullSchedule         `yaml:"full,omitempty"         mapstructure:"full"`
	Differential *DifferentialSchedule `yaml:"differential,omitempty" mapstructure:"differential"`
	Log          *LogSchedule          `yaml:"log,omitempty"          mapstructure:"log"`
}

type FullSchedule struct {
	Frequency  string    `yaml:"frequency"  mapstructure:"frequency"`
	Time       string    `yaml:"time"       mapstructure:"time"`
	Interval   int32     `yaml:"interval"   mapstructure:"interval"`
	ParsedTime time.Time `yaml:"-"          mapstructure:"-"`
}

type OperationalRetention struct {
	Count int32 `yaml:"count" mapstructure:"count"`
}

type DifferentialSchedule struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type LogSchedule struct {
	FrequencyMinutes int `yaml:"frequency_minutes" mapstructure:"frequency_minutes"`
}
type AzureRecoveryVMPolicy struct {
	ResourceGroup string
	VaultName     string
	PolicyName    string
	BackupTime    time.Time
	Retention     RetentionSpec

	InstantRPRetentionDays       int32
	InstantRPResourceGroupPrefix string
	InstantRPResourceGroupSuffix string
}

type AzureRecoveryDiskPolicy struct {
	ResourceGroup               string
	VaultName                   string
	PolicyName                  string
	BackupIntervalHours         int32
	RetentionDays               int32
	SnapshotResourceGroupPrefix string
	SnapshotResourceGroupSuffix string
}
