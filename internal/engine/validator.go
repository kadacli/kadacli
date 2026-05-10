package engine

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// ValidateTarget enforces per-resource-type rules from the spec capability matrix.
// It runs after the resolver has produced a ResolvedTarget.
func ValidateTarget(rt models.ResolvedTarget) []error {
	var errs []error
	targetType := rt.Target.Type
	sched := rt.Policy.Spec.Schedule

	switch targetType {
	case "azure_vm":
		if len(rt.Target.DiskSelection) == 0 && !rt.Target.IncludeFutureDisks {
			errs = append(errs, fmt.Errorf(
				"target %s/%s: invalid disk selection: disk_selection is empty and include_future_disks is false",
				rt.Target.ResourceGroup, rt.Target.ResourceName,
			))
		}

	case "azure_sql":
		// differential schedule is not supported by Azure SQL Database API.
		if sched.Differential != nil && sched.Differential.Enabled {
			errs = append(errs, fmt.Errorf(
				"target %s/%s: schedule.differential is not supported for azure_sql",
				rt.Target.ResourceGroup, rt.Target.ResourceName,
			))
		}
		// disk_selection is invalid for azure_sql.
		if len(rt.Target.DiskSelection) > 0 {
			errs = append(errs, fmt.Errorf(
				"target %s/%s: disk_selection is only valid for azure_vm",
				rt.Target.ResourceGroup, rt.Target.ResourceName,
			))
		}

	case "azure_sql_vm":
		// disk_selection is invalid for azure_sql_vm.
		if len(rt.Target.DiskSelection) > 0 {
			errs = append(errs, fmt.Errorf(
				"target %s/%s: disk_selection is only valid for azure_vm",
				rt.Target.ResourceGroup, rt.Target.ResourceName,
			))
		}

	case "azure_disk":
		// azure_disk uses Backup Vault (not Recovery Services Vault).
		// Differential and log schedules are not supported.
		if sched.Differential != nil && sched.Differential.Enabled {
			errs = append(errs, fmt.Errorf(
				"target %s/%s: schedule.differential is not supported for azure_disk",
				rt.Target.ResourceGroup, rt.Target.ResourceName,
			))
		}
		if sched.Log != nil {
			errs = append(errs, fmt.Errorf(
				"target %s/%s: schedule.log is not supported for azure_disk",
				rt.Target.ResourceGroup, rt.Target.ResourceName,
			))
		}
		if len(rt.Target.DiskSelection) > 0 {
			errs = append(errs, fmt.Errorf(
				"target %s/%s: disk_selection is only valid for azure_vm",
				rt.Target.ResourceGroup, rt.Target.ResourceName,
			))
		}

	default:
		errs = append(errs, fmt.Errorf(
			"target %s/%s: unsupported type %q",
			rt.Target.ResourceGroup, rt.Target.ResourceName, targetType,
		))
	}

	// log.frequency_minutes must be 60–720 (Azure API constraint).
	if sched.Log != nil {
		mins := sched.Log.FrequencyMinutes
		if mins < 60 || mins > 720 {
			errs = append(errs, fmt.Errorf(
				"schedule.log.frequency_minutes must be between 60 and 720, got %d", mins,
			))
		}
	}

	// Retention counts must be positive where set.
	ret := rt.Policy.Spec.Retention
	if ret.Daily != nil && ret.Daily.Count <= 0 {
		errs = append(errs, fmt.Errorf("retention.daily.count must be a positive integer"))
	}
	if ret.Weekly != nil && ret.Weekly.Count <= 0 {
		errs = append(errs, fmt.Errorf("retention.weekly.count must be a positive integer"))
	}
	if ret.Monthly != nil && ret.Monthly.Count <= 0 {
		errs = append(errs, fmt.Errorf("retention.monthly.count must be a positive integer"))
	}
	if ret.Yearly != nil && ret.Yearly.Count <= 0 {
		errs = append(errs, fmt.Errorf("retention.yearly.count must be a positive integer"))
	}

	// Full schedule frequency must be daily or weekly.
	if sched.Full != nil {
		switch sched.Full.Frequency {
		case "daily", "weekly", "hourly":
			// valid
		default:
			errs = append(errs, fmt.Errorf(
				"schedule.full.frequency must be hourly, daily or weekly, got %q", sched.Full.Frequency,
			))
		}
	}

	return errs
}
