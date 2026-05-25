package audit

import (
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// DetectDrift compares the desired policy against the live Azure configuration for
// a resource and returns a list of FieldDiffs describing the divergence.
// TODO: implement for 1.0.0
func DetectDrift(desired models.Policy, resource models.Resource) []models.FieldDiff {
	return nil
}
