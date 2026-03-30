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

func Resolve(policies []models.RawPolicy) []models.BackupPolicy {

	var backupPolicies []models.BackupPolicy

	for _, policy := range policies {
		fullPolicy := merge(policy)

		// Call parser.go ValidatePolicy
		errs := ValidatePolicy(fullPolicy)

		var numOfErrors int = 0
		for _, err := range errs {
			if err != nil {
				numOfErrors++
				fmt.Printf("Invalid field: %s \n", err)
			}
		}
		// Guard-clause
		if numOfErrors > 0 {
			// TODO Print the file path
			panic(fmt.Errorf("You have %d critical errors in your config file with the metadata.name: %s", numOfErrors, policy.Metadata.Name))
		}
		// After guard clause (the filled RawPolicy is valid)
		backupPolicies = append(backupPolicies, construct(fullPolicy))
	}

	return backupPolicies

}

// merge function
// Takes a RawPolicy, looks for non-set fields
// Fills non-set fields with defaults
// Does a simple field-by-field walk
// Returns a "full" RawPolicy
// TODO: Make extensible to handle more future fields
func merge(rawPolicy models.RawPolicy) models.RawPolicy {

	var fullPolicy models.RawPolicy

	if rawPolicy.Metadata.Name == "" {
		rawPolicy.Metadata.Name = models.DefaultMetadata.Name
	}

	if rawPolicy.Metadata.Environment == "" {
		rawPolicy.Metadata.Environment = models.DefaultMetadata.Environment
	}

	// If no schedule is set, copy the entire default schedule
	if rawPolicy.Spec.Schedule.Full.Cron == "" && fullPolicy.Spec.Schedule.Incremental.Cron == "" {
		rawPolicy.Spec.Schedule = models.DefaultSchedule
	}

	if rawPolicy.Spec.Timezone == "" {
		rawPolicy.Spec.Timezone = models.DefaultTimezone
	}

	if rawPolicy.Spec.Retention.Daily == 0 &&
		rawPolicy.Spec.Retention.Weekly == 0 &&
		rawPolicy.Spec.Retention.Monthly == 0 {
		rawPolicy.Spec.Retention = models.DefaultRetention
	}

	if len(rawPolicy.Spec.StorageProviders) == 0 {
		rawPolicy.Spec.StorageProviders =
			append(rawPolicy.Spec.StorageProviders, models.DefualtStorageProvider)
	}

	fullPolicy = rawPolicy

	return fullPolicy
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
