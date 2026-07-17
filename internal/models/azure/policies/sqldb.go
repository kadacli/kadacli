package policies

type AzureSqlDbSpec struct {
	Redundancy         string             `yaml:"redundancy" validate:"oneof=Local Zone Geo GeoZone"`
	ShortTermRetention AzureSqlDbSTR      `yaml:"short_term_retention" validate:"required"`
	LongTermRetention  AzureSqlDbLtr      `yaml:"long_term_retention"`
	Schedule           AzureSqlDbSchedule `yaml:"schedule" validate:"required"`
}

type AzureSqlDbSchedule struct {
	Frequency string `yaml:"frequency" validate:"required,oneof=P1W P2W P4W P1M"`
	Time      string `yaml:"time"      validate:"required,datetime=15:04"`
	Timezone  string `yaml:"timezone"  validate:"omitempty"`
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
