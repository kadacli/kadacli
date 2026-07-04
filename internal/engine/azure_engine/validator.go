package azure_engine

// internal/engine/azure/validator.go

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine/utils_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure/policies"
)

// TODOs:
// Implement AzureDisks, AzureSQLVM, AzureSqlDb

// validates the yaml
var validate = validator.New()

func ValidateAzure(policy models.Policy, sourcePath string) error {
	var resource azure.AzureResource
	if err := policy.Resource.Decode(&resource); err != nil {
		return fmt.Errorf("%s: decoding resource block: %w", sourcePath, err)
	}

	switch resource.Type {
	case "azure_vm":
		return validateAzureVM(policy, resource, sourcePath)
	case "azure_disk":
		return validateAzureDisk(policy, resource, sourcePath)
	default:
		return fmt.Errorf("%s: resource.type %q is not yet supported", sourcePath, resource.Type)
	}
}

func validateAzureDisk(policy models.Policy, resource azure.AzureResource, sourcePath string) error {
	if err := validate.Struct(resource); err != nil {
		return utils_engine.WrapValidationError(err, sourcePath)
	}

	var spec policies.AzureDiskSpec
	if err := policy.Spec.Decode(&spec); err != nil {
		return fmt.Errorf("%s: decoding spec block: %w", sourcePath, err)
	}
	if err := validate.Struct(spec); err != nil {
		return utils_engine.WrapValidationError(err, sourcePath)
	}
	return validateAzureDiskRules(spec, sourcePath)
}

func validateAzureDiskRules(spec policies.AzureDiskSpec, sourcePath string) error {
	// Rule 1: If AzureDiskRetention == false => retentionRules must be set
	if !spec.Retention.Default && len(spec.Retention.RetentionRules) == 0 {
		return fmt.Errorf("RetentionRules required if 'default' == false")
	}
	// Rule 2: validate the AzureDiskRetentionRules if AzureDiskSchedule.Frequency == daily => RetentionRule must be weekly
	if len(spec.Retention.RetentionRules) > 0 {
		if spec.Schedule.Frequency == "daily" {
			for _, rule := range spec.Retention.RetentionRules {
				if rule.RetentionRule != "weekly" {
					return fmt.Errorf("%s: RetentionRule must be 'weekly' when schedule.frequency == 'daily'", sourcePath)
				}
			}
		}
	}

	// Rule 3: If schedule.frequency == hourly => validate AzureDiskSchedule.Interval
	if spec.Schedule.Frequency == "hourly" {
		if spec.Schedule.Interval <= 0 {
			return fmt.Errorf("%s: schedule.interval must be an integer with one of the following values: [1, 2, 4, 6, 8 12], when schedule.frequency == 'hourly'", sourcePath)
		}
	}

	// Rule 4: If schedule.frequency == daily => validate AzureDiskSchedule.Time
	if spec.Schedule.Frequency == "daily" {
		if spec.Schedule.Time == "" {
			return fmt.Errorf("%s: schedule.time is required when schedule.frequency == 'daily'", sourcePath)
		}
		// Optional: Validate time format (HH:MM)
		if _, err := time.Parse("15:04", spec.Schedule.Time); err != nil {
			return fmt.Errorf("%s: schedule.time must be in 'HH:MM' format", sourcePath)
		}
	}

	return nil
}

func validateAzureVM(policy models.Policy, resource azure.AzureResource, sourcePath string) error {
	if err := validate.Struct(resource); err != nil {
		return utils_engine.WrapValidationError(err, sourcePath)
	}

	var spec policies.AzureVMSpec
	if err := policy.Spec.Decode(&spec); err != nil {
		return fmt.Errorf("%s: decoding spec block: %w", sourcePath, err)
	}
	if err := validate.Struct(spec); err != nil {
		return utils_engine.WrapValidationError(err, sourcePath)
	}

	return validateAzureVMRules(spec, sourcePath)
}

// validateAzureVMRules holds the logic-based checks that go-playground
// tags cannot express on their own.
func validateAzureVMRules(spec policies.AzureVMSpec, sourcePath string) error {
	if spec.Schedule.Full == nil {
		return fmt.Errorf("%s: schedule.full is required", sourcePath)
	}
	full := spec.Schedule.Full

	// Rule 1 — if full backup frequency is weekly, a day must be specified
	if full.Frequency == "weekly" && full.Day == "" {
		return fmt.Errorf("%s: schedule.full.day is required when frequency is 'weekly'", sourcePath)
	}

	// Rule 2 — if a weekly retention exists, its day must align with
	// the full backup day when the full backup is weekly
	if spec.Retention.Weekly != nil && full.Frequency == "weekly" {
		if spec.Retention.Weekly.Day != full.Day {
			return fmt.Errorf(
				"%s: retention.weekly.day (%s) must match schedule.full.day (%s) for weekly backups",
				sourcePath, spec.Retention.Weekly.Day, full.Day,
			)
		}
	}

	// Rule 3 — yearly retention requires a weekly or monthly anchor to exist
	if spec.Retention.Yearly != nil && spec.Retention.Weekly == nil && spec.Retention.Monthly == nil {
		return fmt.Errorf(
			"%s: retention.yearly requires either a weekly or monthly retention rule to anchor to",
			sourcePath,
		)
	}

	return nil
}

// ValidateAzureAssignment validates the top-level and provider-specific fields
// of an Azure assignment. Returns the first error encountered.
func ValidateAzureAssignment(assignment models.Assignment, sourcePath string) error {
	errs := ParseAssignment(assignment)
	if len(errs) == 0 {
		return nil
	}
	msg := fmt.Sprintf("%s: assignment validation failed:", sourcePath)
	for _, e := range errs {
		msg += fmt.Sprintf("\n  - %s", e.Error())
	}
	return fmt.Errorf("%s", msg)
}
