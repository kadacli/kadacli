package policies

type AzureSqlDbSpec struct {
	Redundancy         string        `yaml:"redundancy" validate:"oneof=Local Zone Geo GeoZone"`
	ShortTermRetention AzureSqlDbSTR `yaml:"short_term_retention" validate:"required"`
	LongTermRetention  AzureSqlDbLtr `yaml:"long_term_retention"`
	// NOTE: Azure SQL Database PITR/LTR backups are automatic and not
	// schedule-configurable via this API, so there is no `schedule` block —
	// unlike azure_vm (AzureSchedule) and azure_disk (AzureDiskSchedule).
}

type AzureSqlDbSTR struct {
	RetentionDays               int `yaml:"retention_days"                validate:"required,min=1,max=35"`
	DifferentialBackupFrequency int `yaml:"differential_backup_frequency" validate:"omitempty,oneof=12 24"`
}

type AzureSqlDbLtr struct {
	WeeklyRetention  int   `yaml:"weekly_retention"  validate:"omitempty,min=1,max=520"`
	MonthlyRetention int   `yaml:"monthly_retention" validate:"omitempty,min=1,max=120"`
	YearlyRetention  int   `yaml:"yearly_retention"  validate:"omitempty,min=1,max=10"`
	WeekOfYear       int   `yaml:"week_of_year"      validate:"omitempty,min=1,max=52"`
	LtrImmutability  *bool `yaml:"ltr_immutability"`
	LockImmutability *bool `yaml:"lock_immutability"`
}
