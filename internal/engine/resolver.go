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

// A BackupPolicy is a validated "RawPolicy"

import (
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

func Resolve(policies []models.RawPolicy) ([]models.BackupPolicy, []error) {

	var backupPolicies []models.BackupPolicy

	for i := range policies {

		policy := policies[i]

		// Call parser.go ValidatePolicy
		errs := ValidatePolicy(policy)

		var numOfErrors int = 0
		for _, err := range errs {
			if err != nil {
				numOfErrors++
			}
		}
		// Guard-clause
		if numOfErrors > 0 {
			return nil, errs
		}
		// After guard clause (the filled RawPolicy is valid)
		for target_n := range policy.Spec.Targets {
			backupPolicies = append(backupPolicies, construct(policy, target_n))
		}
	}

	return backupPolicies, nil

}

// Construct function
// Inputs: A valid "RawPolicy"
// Returns: A BackupPolicy
// Becuase the RawPolicy is valid, we can copy field-by-field
func construct(validPolicy models.RawPolicy, target_n int) models.BackupPolicy {
	var backupPolicy models.BackupPolicy

	backupPolicy.ApiVersion = validPolicy.ApiVersion
	backupPolicy.Kind = validPolicy.Kind
	backupPolicy.Metadata = validPolicy.Metadata
	backupPolicy.SpecialSpec.Target = validPolicy.Spec.Targets[target_n]
	backupPolicy.SpecialSpec.Timezone = validPolicy.Spec.Timezone
	backupPolicy.SpecialSpec.StorageProvider = validPolicy.Spec.StorageProvider

	return backupPolicy

}
