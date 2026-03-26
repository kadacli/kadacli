package models

// The defaults define the default values
var DefaultSchedule = Schedule{
	Full:        CronSchedule{Cron: "0 2 * * 0"},
	Incremental: CronSchedule{Cron: "0 2 * * 1-6"},
}

var DefaultRetention = Retention{
	Daily:   7,
	Weekly:  4,
	Monthly: 12,
}

var DefaultMetadata = Metadata{
	Environment: "Production",
}

// TODO: Load automatically based on system that executes the policy
var DefaultTimezone = "Europe/Copenhagen"

var DefaultStorage = []Storage{
	{
		Primary: StorageProvider{
			Provider:   "azure",
			Vault:      "backup-vault-prod",
			Redundancy: "GRS",
		},
	},
}
