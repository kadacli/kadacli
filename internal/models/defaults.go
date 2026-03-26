package models

// The defaults defines the default values
var DefaultSchedule = Schedule{
	Full:        CronSchedule{Cron: "0 2 * * 0"},
	Incremental: CronSchedule{Cron: "0 2 * * 1-6"},
}
