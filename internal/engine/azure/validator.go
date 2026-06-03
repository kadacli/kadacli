// internal/engine/validator.go
package engine

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/providers/azure/azure_policies"
)

var validate = validator.New()

// ValidatePolicy runs the full validation pipeline for one policy file.
func ValidatePolicy(policy models.Policy, sourcePath string) error {
	// Stage 1 — validate the provider-free top level
	if err := validate.Struct(policy); err != nil {
		return wrapValidationError(err, sourcePath)
	}

	// Stage 2 — decode and validate the provider-specific blocks
	switch policy.Provider {
	case "azure":
		return validateAzure(policy, sourcePath)
	default:
		return fmt.Errorf("%s: provider %q is not yet supported", sourcePath, policy.Provider)
	}
}

func validateAzure(policy models.Policy, sourcePath string) error {
	var resource AzureModel.AzureResource
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

func validateAzureVM(policy models.Policy, resource AzureModel.AzureResource, sourcePath string) error {
	if err := validate.Struct(resource); err != nil {
		return wrapValidationError(err, sourcePath)
	}

	var spec azure_policies.AzureVMSpec
	if err := policy.Spec.Decode(&spec); err != nil {
		return fmt.Errorf("%s: decoding spec block: %w", sourcePath, err)
	}
	if err := validate.Struct(spec); err != nil {
		return wrapValidationError(err, sourcePath)
	}

	if err := validateAzureVMRules(spec, sourcePath); err != nil {
		return err
	}

	return nil
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

	// Rule 2 — if a weekly retention exists, its day should align with
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

// wrapValidationError turns raw validator output into readable messages.
func wrapValidationError(err error, sourcePath string) error {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		msg := fmt.Sprintf("%s: validation failed:\n", sourcePath)
		for _, e := range verrs {
			msg += fmt.Sprintf("  - %s: %q is not valid (rule: %s)\n",
				e.Namespace(), e.Value(), e.Tag())
		}
		return errors.New(msg)
	}
	return fmt.Errorf("%s: %w", sourcePath, err)
}
