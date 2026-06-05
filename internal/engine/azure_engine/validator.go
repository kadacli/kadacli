package azure_engine

// internal/engine/azure/validator.go

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine/utils_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models/azure_policies"
)

// TODOs:
// Implement AzureDisks, AzureSQLVM, AzureSqlDb

// validates the yaml
var validate = validator.New()

func ValidateAzure(policy models.Policy, sourcePath string) error {
	var resource azure_models.AzureResource
	if err := policy.Resource.Decode(&resource); err != nil {
		return fmt.Errorf("%s: decoding resource block: %w", sourcePath, err)
	}

	switch resource.ResourceKind {
	case "vm":
		return validateAzureVM(policy, resource, sourcePath)
	default:
		return fmt.Errorf("%s: azure resourceKind %q is not yet supported", sourcePath, resource.ResourceKind)
	}
}

func validateAzureVM(policy models.Policy, resource azure_models.AzureResource, sourcePath string) error {
	if err := validate.Struct(resource); err != nil {
		return utils_engine.WrapValidationError(err, sourcePath)
	}

	var spec azure_policies.AzureVMSpec
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
func validateAzureVMRules(spec azure_policies.AzureVMSpec, sourcePath string) error {
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
