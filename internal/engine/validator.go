package engine

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine/utils_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/go-playground/validator/v10"
)

// engine.validator.go validates the top-level (provider agnostic) part of the policies and assignments.
// It also contains any helper functions that all other engines might use

var validate = validator.New()

// ValidatePolicy validates the top-level and determines the provider
func ValidatePolicy(policy models.Policy, sourcePath string) error {
	if err := validate.Struct(policy); err != nil {
		return utils_engine.WrapValidationError(err, sourcePath)
	}

	switch policy.Provider {
	case "azure":
		return azure_engine.ValidateAzure(policy, sourcePath)
	default:
		return fmt.Errorf("%s: provider %q is not yet supported", sourcePath, policy.Provider)
	}
}

// ValidateAssignment validates the top-level and provider-specific fields of an assignment.
func ValidateAssignment(assignment models.Assignment, sourcePath string) error {
	if err := validate.Struct(assignment); err != nil {
		return utils_engine.WrapValidationError(err, sourcePath)
	}

	switch assignment.Provider {
	case "azure":
		return azure_engine.ValidateAzureAssignment(assignment, sourcePath)
	default:
		return fmt.Errorf("%s: provider %q is not yet supported", sourcePath, assignment.Provider)
	}
}
