package engine

// In internal/engine/resolver.go implement
// Resolve(policies []RawPolicy, defaults []PolicyDefaults) []BackupPolicy)
// For each policy with a non-empty Metadata.Inherits field find
// the matching defaults and deep merge the PartialSpec
// (defaults fill gaps, policy-level values always win).
//
// This must be a pure function — no file I/O.

// Notes to self:
// Should have access to the default values
// Recieves a RawPolicy and fills in any gaps
// A gap is a field that is not filled out

// A BackupPolicy is a "RawPolicy" that has no gaps

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

func Resolve(policies []models.RawPolicy) ([]models.BackupPolicy, []error) {

	var backupPolicies []models.BackupPolicy

	for i, _ := range policies {

		policy := policies[i]

		// Call parser.go ValidatePolicy
		errs := ValidatePolicy(policy)

		var numOfErrors int = 0
		for _, err := range errs {
			if err != nil {
				numOfErrors++
				fmt.Printf("Invalid field: %s \n", err)
			}
		}
		// Guard-clause
		if numOfErrors > 0 {

			return nil, errs
		}
		// After guard clause (the filled RawPolicy is valid)
		backupPolicies = append(backupPolicies, construct(policy))
	}

	return backupPolicies, nil

}

// Construct function
// Inputs: A valid "RawPolicy"
// Returns: A BackupPolicy
// Becuase the RawPolicy is valid, we can copy field-by-field
func construct(validPolicy models.RawPolicy) models.BackupPolicy {
	var backupPolicy models.BackupPolicy

	backupPolicy.ApiVersion = validPolicy.ApiVersion
	backupPolicy.Kind = validPolicy.Kind
	backupPolicy.Metadata = validPolicy.Metadata
	backupPolicy.Spec = validPolicy.Spec

	return backupPolicy

}
