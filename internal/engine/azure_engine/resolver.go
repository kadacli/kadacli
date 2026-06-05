// internal/engine/azure_engine/resolver.go
package azure_engine

import (
	"fmt"
	"time"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	azure_models "github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models"
	azure_assignments "github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models/azure_assignments"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models/azure_policies"
)

// ResolveVMs matches each assignment target to its named policy and returns
// a flat slice of ResolvedVMs with all data merged and validated.
func ResolveVMs(policies []models.Policy, assignments []models.Assignment) ([]azure_models.ResolvedVM, []error) {
	var errs []error

	// Validate and index policies by name
	policyIndex := make(map[string]models.Policy, len(policies))
	for _, p := range policies {
		if parseErrs := ParsePolicy(p); len(parseErrs) > 0 {
			for _, e := range parseErrs {
				errs = append(errs, fmt.Errorf("policy %q: %w", p.Metadata.Name, e))
			}
			continue
		}
		if _, exists := policyIndex[p.Metadata.Name]; exists {
			errs = append(errs, fmt.Errorf("duplicate policy name %q", p.Metadata.Name))
			continue
		}
		policyIndex[p.Metadata.Name] = p
	}

	// Validate assignments
	for _, a := range assignments {
		if assignErrs := ParseAssignment(a); len(assignErrs) > 0 {
			for _, e := range assignErrs {
				errs = append(errs, fmt.Errorf("assignment %q: %w", a.Metadata.Name, e))
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Resolve each assignment target against its policy
	var resolved []azure_models.ResolvedVM
	for _, a := range assignments {
		// Decode the assignment resource block
		var assignmentResource azure_models.AzureResource
		if err := a.Resource.Decode(&assignmentResource); err != nil {
			errs = append(errs, fmt.Errorf("assignment %q: decoding resource block: %w", a.Metadata.Name, err))
			continue
		}

		// Decode the assignments block
		var vmConfig azure_assignments.VMConfig
		if err := a.Assignments.Decode(&vmConfig); err != nil {
			errs = append(errs, fmt.Errorf("assignment %q: decoding assignments block: %w", a.Metadata.Name, err))
			continue
		}

		for _, pa := range vmConfig.Assignments {
			policy, ok := policyIndex[pa.Policy]
			if !ok {
				errs = append(errs, fmt.Errorf(
					"assignment %q references unknown policy %q", a.Metadata.Name, pa.Policy,
				))
				continue
			}

			// Decode the policy spec
			var spec azure_policies.AzureVMSpec
			if err := policy.Spec.Decode(&spec); err != nil {
				errs = append(errs, fmt.Errorf(
					"assignment %q policy %q: decoding spec: %w", a.Metadata.Name, pa.Policy, err,
				))
				continue
			}

			// Parse the backup time once here so downstream consumers get a
			// ready-to-use value
			if spec.Schedule.Full != nil {
				t, err := parseBackupTime(spec.Schedule.Full.Time)
				if err != nil {
					errs = append(errs, fmt.Errorf(
						"assignment %q policy %q: invalid schedule.full.time %q: %w",
						a.Metadata.Name, pa.Policy, spec.Schedule.Full.Time, err,
					))
					continue
				}
				spec.Schedule.Full.ParsedTime = t
			}

			for _, target := range pa.Targets {
				resolved = append(resolved, azure_models.ResolvedVM{
					SubscriptionID:     assignmentResource.SubscriptionID,
					ResourceGroup:      target.ResourceGroup,
					ResourceName:       target.ResourceName,
					VaultName:          assignmentResource.VaultName,
					IncludeFutureDisks: target.IncludeFutureDisks,
					PolicyName:         pa.Policy,
					Resource:           assignmentResource,
					Spec:               spec,
				})
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return resolved, nil
}

// parseBackupTime parses a "15:04" UTC time string into a time.Time.
// The date component is set to the zero date; only the time matters to Azure.
func parseBackupTime(s string) (time.Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected HH:MM format, got %q", s)
	}
	return t.UTC(), nil
}
