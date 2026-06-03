package engine

// 31/05/2026: Not needed I suppose? //LBH

import (
	"fmt"
	"slices"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// ParsePolicy validates top-level structural requirements of a Policy
// before it enters the resolver. Per-resource validation is in validator.go.
func ParsePolicy(p models.Policy) []error {
	var errs []error

	possibleDayOfWeekValues := armrecoveryservicesbackup.PossibleDayOfWeekValues()
	possibleMonthOfYearValues := armrecoveryservicesbackup.PossibleMonthOfYearValues()
	possibleWeekOfMonthValues := armrecoveryservicesbackup.PossibleWeekOfMonthValues()

	// --- Metadata ---
	if p.APIVersion == "" {
		errs = append(errs, fmt.Errorf("apiVersion is required"))
	} else if p.APIVersion != "backupctl/v1" {
		errs = append(errs, fmt.Errorf("apiVersion must be backupctl/v1, got %q", p.APIVersion))
	}
	if p.Kind != "Policy" {
		errs = append(errs, fmt.Errorf("kind must be Policy, got %q", p.Kind))
	}
	if p.Metadata.Name == "" {
		errs = append(errs, fmt.Errorf("metadata.name is required"))
	}
	if p.Metadata.Owner == "" {
		errs = append(errs, fmt.Errorf("metadata.owner is required"))
	}

	// --- Spec ---
	if p.Spec.Timezone == "" {
		errs = append(errs, fmt.Errorf("spec.timezone is required"))
	}
	if p.Spec.Vault.Name == "" {
		errs = append(errs, fmt.Errorf("spec.vault.name is required"))
	}
	if p.Spec.Vault.ResourceGroup == "" {
		errs = append(errs, fmt.Errorf("spec.vault.resource_group is required"))
	}

	// --- Schedule ---
	if p.Spec.Schedule.Full == nil {
		errs = append(errs, fmt.Errorf("spec.schedule.full is required"))
	} else {
		switch p.Spec.Schedule.Full.Frequency {
		case "daily", "weekly":
			if p.Spec.Schedule.Full.Time == "" {
				errs = append(errs, fmt.Errorf("spec.schedule.full.time is required for daily/weekly frequency"))
			} else if _, err := time.Parse("15:04", p.Spec.Schedule.Full.Time); err != nil {
				errs = append(errs, fmt.Errorf("spec.schedule.full.time must be in HH:MM format, got %q", p.Spec.Schedule.Full.Time))
			}
		case "hourly":
			validIntervals := []int32{1, 2, 4, 6, 8, 12}
			if !slices.Contains(validIntervals, p.Spec.Schedule.Full.Interval) {
				errs = append(errs, fmt.Errorf(
					"spec.schedule.full.interval must be one of %v for hourly frequency, got %d",
					validIntervals, p.Spec.Schedule.Full.Interval,
				))
			}
			if p.Spec.Retention.Operational == nil || p.Spec.Retention.Operational.Count <= 0 {
				errs = append(errs, fmt.Errorf("spec.retention.operational.count is required for hourly (disk) policies"))
			} else if p.Spec.Retention.Operational.Count > 30 {
				errs = append(errs, fmt.Errorf(
					"spec.retention.operational.count must be between 1 and 30, got %d",
					p.Spec.Retention.Operational.Count,
				))
			}
		case "":
			errs = append(errs, fmt.Errorf("spec.schedule.full.frequency is required"))
		default:
			errs = append(errs, fmt.Errorf(
				"spec.schedule.full.frequency must be daily, weekly, or hourly, got %q",
				p.Spec.Schedule.Full.Frequency,
			))
		}
	}

	if p.Spec.Schedule.Log != nil {
		mins := p.Spec.Schedule.Log.FrequencyMinutes
		if mins < 60 || mins > 720 {
			errs = append(errs, fmt.Errorf(
				"spec.schedule.log.frequency_minutes must be between 60 and 720, got %d", mins,
			))
		}
	}

	// --- Retention (VM policies) ---
	ret := p.Spec.Retention
	if p.Spec.Schedule.Full != nil && p.Spec.Schedule.Full.Frequency != "hourly" {
		if ret.Daily == nil || ret.Daily.Count <= 0 {
			errs = append(errs, fmt.Errorf("spec.retention.daily.count is required and must be a positive integer"))
		}
		if ret.Weekly != nil {
			if ret.Weekly.Count <= 0 {
				errs = append(errs, fmt.Errorf("spec.retention.weekly.count must be a positive integer"))
			}
			if !slices.Contains(possibleDayOfWeekValues, armrecoveryservicesbackup.DayOfWeek(ret.Weekly.Day)) {
				errs = append(errs, fmt.Errorf(
					"spec.retention.weekly.day %q is invalid, must be one of %v",
					ret.Weekly.Day, possibleDayOfWeekValues,
				))
			}
		}
		if ret.Monthly != nil && ret.Monthly.Count <= 0 {
			errs = append(errs, fmt.Errorf("spec.retention.monthly.count must be a positive integer"))
		}
		if ret.Yearly != nil {
			if ret.Yearly.Count <= 0 {
				errs = append(errs, fmt.Errorf("spec.retention.yearly.count must be a positive integer"))
			}
			if !slices.Contains(possibleMonthOfYearValues, armrecoveryservicesbackup.MonthOfYear(ret.Yearly.Month)) {
				errs = append(errs, fmt.Errorf(
					"spec.retention.yearly.month %q is invalid, must be one of %v",
					ret.Yearly.Month, possibleMonthOfYearValues,
				))
			}
			if !slices.Contains(possibleDayOfWeekValues, armrecoveryservicesbackup.DayOfWeek(ret.Weekly.Day)) {
				errs = append(errs, fmt.Errorf(
					"spec.retention.yearly.day %q is invalid, must be one of %v",
					ret.Yearly.Day, possibleDayOfWeekValues,
				))
			}
			if !slices.Contains(possibleWeekOfMonthValues, armrecoveryservicesbackup.WeekOfMonth(ret.Yearly.Week)) {
				errs = append(errs, fmt.Errorf(
					"spec.retention.yearly.week %q is invalid, must be one of %v",
					ret.Yearly.Week, possibleWeekOfMonthValues,
				))
			}
		}

		// --- InstantRP (VM only) ---
		if p.Spec.InstantRPRetentionDays < 0 {
			errs = append(errs, fmt.Errorf("spec.instant_rp_retention_days must be a positive integer"))
		}
	}

	return errs
}

// ParseAssignment validates top-level structural requirements of an Assignment.
func ParseAssignment(a models.Assignment) []error {
	var errs []error

	if a.APIVersion == "" {
		errs = append(errs, fmt.Errorf("apiVersion is required"))
	} else if a.APIVersion != "backupctl/v1" {
		errs = append(errs, fmt.Errorf("apiVersion must be backupctl/v1, got %q", a.APIVersion))
	}
	if a.Kind != "Assignment" {
		errs = append(errs, fmt.Errorf("kind must be Assignment, got %q", a.Kind))
	}
	if a.Metadata.Name == "" {
		errs = append(errs, fmt.Errorf("metadata.name is required"))
	}
	if a.Metadata.SubscriptionID == "" {
		errs = append(errs, fmt.Errorf("metadata.subscriptionId is required"))
	}
	if a.Metadata.Owner == "" {
		errs = append(errs, fmt.Errorf("metadata.owner is required"))
	}
	if len(a.Spec.Assignments) == 0 {
		errs = append(errs, fmt.Errorf("spec.assignments must not be empty"))
	}
	for i, pa := range a.Spec.Assignments {
		if pa.Policy == "" {
			errs = append(errs, fmt.Errorf("spec.assignments[%d].policy is required", i))
		}
		if len(pa.Targets) == 0 {
			errs = append(errs, fmt.Errorf("spec.assignments[%d].targets must not be empty", i))
		}
		for j, t := range pa.Targets {
			if t.Type == "" {
				errs = append(errs, fmt.Errorf("spec.assignments[%d].targets[%d].type is required", i, j))
			}
			if t.ResourceGroup == "" {
				errs = append(errs, fmt.Errorf("spec.assignments[%d].targets[%d].resource_group is required", i, j))
			}
			if t.ResourceName == "" {
				errs = append(errs, fmt.Errorf("spec.assignments[%d].targets[%d].resource_name is required", i, j))
			}
		}
	}

	return errs
}
