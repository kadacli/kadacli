package engine

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// Resolve matches each AssignmentTarget to its named Policy and returns a flat
// slice of ResolvedTargets with overrides merged in. Policy-level values win
// over overrides for fields not explicitly overridden.
func Resolve(policies []models.Policy, assignments []models.Assignment) ([]models.ResolvedTarget, []error) {
	var errs []error

	// Validate and index policies by name.
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

	// Validate assignments.
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

	var resolved []models.ResolvedTarget
	for _, a := range assignments {
		for _, pa := range a.Spec.Assignments {
			basePolicy, ok := policyIndex[pa.Policy]
			if !ok {
				errs = append(errs, fmt.Errorf(
					"assignment %q references unknown policy %q", a.Metadata.Name, pa.Policy,
				))
				continue
			}
			for _, target := range pa.Targets {
				rt := models.ResolvedTarget{
					SubscriptionID: a.Metadata.SubscriptionID,
					PolicyName:     pa.Policy,
					Target:         target,
					Policy:         applyOverrides(basePolicy, target.Overrides),
				}
				if valErrs := ValidateTarget(rt); len(valErrs) > 0 {
					errs = append(errs, valErrs...)
					continue
				}
				resolved = append(resolved, rt)
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return resolved, nil
}

// applyOverrides merges target-level overrides onto a copy of the base policy.
// The base policy values are preserved for fields not explicitly overridden.
func applyOverrides(base models.Policy, overrides *models.PolicyOverrides) models.Policy {
	if overrides == nil {
		return base
	}
	merged := base
	if overrides.Retention != nil {
		r := *overrides.Retention
		if r.Daily != nil {
			merged.Spec.Retention.Daily = r.Daily
		}
		if r.Weekly != nil {
			merged.Spec.Retention.Weekly = r.Weekly
		}
		if r.Monthly != nil {
			merged.Spec.Retention.Monthly = r.Monthly
		}
		if r.Yearly != nil {
			merged.Spec.Retention.Yearly = r.Yearly
		}
	}
	if overrides.Schedule != nil {
		s := *overrides.Schedule
		if s.Full != nil {
			merged.Spec.Schedule.Full = s.Full
		}
		if s.Differential != nil {
			merged.Spec.Schedule.Differential = s.Differential
		}
		if s.Log != nil {
			merged.Spec.Schedule.Log = s.Log
		}
	}
	return merged
}
