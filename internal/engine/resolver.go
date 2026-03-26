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

import "github.com/dbjorklund/backup-policy-as-code/internal/models"

func Resolve(policies []models.RawPolicy) []models.BackupPolicy {

}
