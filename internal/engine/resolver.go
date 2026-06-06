package engine

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// Resolve merges all validated assignments against their policies and returns
// a flat slice of ResolvedTargets ready for diff and apply.
// Policies and assignments from different providers are handled independently.
func Resolve(pathPolicies []models.PathPolicy, pathAssignments []models.PathAssignment) ([]models.ResolvedTarget, []error) {
	var errs []error
	var resolved []models.ResolvedTarget

	// Split into provider buckets
	azurePolicies := make([]models.Policy, 0)
	azureAssignments := make([]models.Assignment, 0)

	for _, pp := range pathPolicies {
		switch pp.Policy.Provider {
		case "azure":
			azurePolicies = append(azurePolicies, pp.Policy)
		default:
			errs = append(errs, fmt.Errorf("resolve: provider %q is not yet supported", pp.Policy.Provider))
		}
	}

	for _, pa := range pathAssignments {
		switch pa.Assignment.Provider {
		case "azure":
			azureAssignments = append(azureAssignments, pa.Assignment)
		default:
			errs = append(errs, fmt.Errorf("resolve: provider %q is not yet supported", pa.Assignment.Provider))
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Azure
	if len(azurePolicies) > 0 || len(azureAssignments) > 0 {
		vms, resolveErrs := azure_engine.ResolveVMs(azurePolicies, azureAssignments)
		if len(resolveErrs) > 0 {
			errs = append(errs, resolveErrs...)
		} else {
			for _, vm := range vms {
				resolved = append(resolved, models.ResolvedTarget{
					Provider: "azure",
					Payload:  vm,
				})
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return resolved, nil
}
