package assignments

type SqlDbConfig struct {
	Assignments []SqlDbAssignment `yaml:"assignments"`
}

type SqlDbAssignment struct {
	Policy  string        `yaml:"policy"`
	Targets []SqlDbTarget `yaml:"targets"`
}

type SqlDbTarget struct {
	ResourceGroup string          `yaml:"resource_group" validate:"required"`
	ServerName    string          `yaml:"server_name"    validate:"required"`
	ResourceName  string          `yaml:"resource_name"  validate:"required"`
	Overrides     *SqlDbOverrides `yaml:"overrides,omitempty"`
}

type SqlDbOverrides struct {
	ShortTermRetention *SqlDbSTROverride `yaml:"short_term_retention,omitempty"`
	LongTermRetention  *SqlDbLTROverride `yaml:"long_term_retention,omitempty"`
}

type SqlDbSTROverride struct {
	RetentionDays               int `yaml:"retention_days,omitempty"`
	DifferentialBackupFrequency int `yaml:"differential_backup_frequency,omitempty"`
}

type SqlDbLTROverride struct {
	WeeklyRetention  string `yaml:"weekly_retention,omitempty"`
	MonthlyRetention string `yaml:"monthly_retention,omitempty"`
	YearlyRetention  string `yaml:"yearly_retention,omitempty"`
	WeekOfYear       int    `yaml:"week_of_year,omitempty"`
}
