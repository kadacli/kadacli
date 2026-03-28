package engine

import "github.com/dbjorklund/backup-policy-as-code/internal/models"

// In internal/engine/parser.go implement ValidatePolicy(policy BackupPolicy, sourcePath string) error.
// Run go-playground/validator on the fully resolved struct.
// If validation fails wrap the error with the source file path so the message
//   reads: policies/production/finance.yaml: metadata.owner is required.
// Runs after resolution so it validates the merged result.

// Checks the required fields "if parent is set"
func ValidatePolicy(filledPolicy models.RawPolicy) (bool, []string) {
	var invalidFields []string
	var allFieldsValid bool = true

	// What does it mean that a field is valid?
	// 1. The field is set if required

	return allFieldsValid, invalidFields
}
