package engine

import (
	"fmt"

	"slices"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// ParsePolicy validates top-level structural requirements of a Policy
// before it enters the resolver. Per-resource validation is in validator.go.
func ParsePolicy(p models.Policy) []error {
	var errs []error
	possibleDayOfWeekValues := armrecoveryservicesbackup.PossibleDayOfWeekValues()
	possibleMonthOfYearValues := armrecoveryservicesbackup.PossibleMonthOfYearValues()

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
	if p.Spec.Timezone == "" {
		errs = append(errs, fmt.Errorf("spec.timezone is required"))
	}
	if p.Spec.Vault.Name == "" {
		errs = append(errs, fmt.Errorf("spec.vault.name is required"))
	}
	if p.Spec.Vault.ResourceGroup == "" {
		errs = append(errs, fmt.Errorf("spec.vault.resource_group is required"))
	}
	// TODO validate the retention policy
	if p.Spec.Retention.DailyCount == 0 {
		errs = append(errs, fmt.Errorf("spec.retention.daily is required"))
	}
	// If there is a weekly retention - make sure the weekday is valid
	if p.Spec.Retention.WeeklyCount != 0 {
		if !slices.Contains(possibleDayOfWeekValues, p.Spec.Retention.WeeklyDay) {
			errs = append(errs, fmt.Errorf("The week day in spec.retention.weekly.day is invalid"))
		}
	}
	// If there is a monthly retention - make sure the month is valid
	if p.Spec.Retention.YearlyCount != 0 {
		if !slices.Contains(possibleMonthOfYearValues, p.Spec.Retention.YearlyMonth) {
			errs = append(errs, fmt.Errorf("The month in spec.retention.yearly.month is invalid"))
		}
	}

	if p.Spec.Schedule.Full.Frequency == "" {
		errs = append(errs, fmt.Errorf("The backup schedule's frequency, spec.schedule.full.frequency is required"))
	}

	if p.Spec.Schedule.Full.Time == "" {
		errs = append(errs, fmt.Errorf("spec.schedule.full.time is required"))
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
