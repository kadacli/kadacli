package models

// TODO: Make one file for each resource

// models package defines the structs used to define the backup policies

// The overall struct. Represents the entire unvalidated configuration file
type RawPolicy struct {
	ApiVersion string   `mapstructure:"apiVersion"`
	Kind       string   `mapstructure:"kind"`
	Metadata   Metadata `mapstructure:"metadata"`
	Spec       Spec     `mapstructure:"spec"`
}

type Metadata struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
	Owner       string `mapstructure:"owner"`
}

type Target struct {
	Type      string        `mapstructure:"type"`
	Selectors AzureSelector `mapstructure:"selectors"`
	Databases []DBTarget    `mapstructure:"databases"` // nil for VM targets
}

type DBTarget struct {
	Name string `mapstructure:"name"`
}

// Might extend to more selectors later
type AzureSelector struct {
	ResourceGroup string `mapstructure:"resource_group"`
	ResourceName  string `mapstructure:"resource_name"`
}
type StorageProvider struct {
	Provider           string  `mapstructure:"provider"`
	Vault              string  `mapstructure:"vault"`
	Resource_group     string  `mapstructure:"resource_group"` // Hard-coded to Azure
	BackupPolicy       string  `mapstructure:"backup_policy"`
	DiskSelection      []int32 `mapstructure:"disk_selection"`
	IncludeFutureDisks bool    `mapstructure:"include_future_disks"`
}

type Spec struct {
	Targets         []Target        `mapstructure:"targets"`
	Timezone        string          `mapstructure:"timezone"`
	StorageProvider StorageProvider `mapstructure:"storage"`
	DBPolicy        *DBPolicy       `mapstructure:"db_policy"` // nil for VM policies
}

type DBPolicy struct {
	Full         FullBackupSchedule  `mapstructure:"full"`
	Differential *DiffBackupSchedule `mapstructure:"differential"` // optional
	Log          LogBackupSchedule   `mapstructure:"log"`
	Retention    DBRetention         `mapstructure:"retention"`
}
type FullBackupSchedule struct {
	Frequency string   `mapstructure:"frequency"` // "Daily" | "Weekly"
	Weekdays  []string `mapstructure:"weekdays"`  // e.g. ["Sunday"]
	Time      string   `mapstructure:"time"`      // "HH:MM" in policy timezone
}

type DiffBackupSchedule struct {
	Enabled  bool     `mapstructure:"enabled"`
	Weekdays []string `mapstructure:"weekdays"`
	Time     string   `mapstructure:"time"`
}

type LogBackupSchedule struct {
	FrequencyMinutes int32 `mapstructure:"frequency_minutes"` // 15–120
}

type DBRetention struct {
	FullBackupDays         int32                `mapstructure:"full_backup_days"`
	DifferentialBackupDays int32                `mapstructure:"differential_backup_days"`
	LogBackupDays          int32                `mapstructure:"log_backup_days"`
	LongTerm               *LongTermDBRetention `mapstructure:"long_term"` // nil = disabled
}

type LongTermDBRetention struct {
	WeeklyRetention  string `mapstructure:"weekly_retention"`  // ISO 8601, e.g. "P4W"
	MonthlyRetention string `mapstructure:"monthly_retention"` // e.g. "P12M"
	YearlyRetention  string `mapstructure:"yearly_retention"`  // e.g. "P5Y"
	WeekOfYear       int32  `mapstructure:"week_of_year"`      // 1–52
}

// The BackupPolicy type
// Structureally identical to RawPolicy, but must only be created after validation
type BackupPolicy struct {
	ApiVersion string   `mapstructure:"apiVersion"`
	Kind       string   `mapstructure:"kind"`
	Metadata   Metadata `mapstructure:"metadata"`
	Spec       Spec     `mapstructure:"spec"`
}
