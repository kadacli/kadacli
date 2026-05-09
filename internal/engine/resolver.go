package engine

import (
	"fmt"
	"time"

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
				merged := applyOverrides(basePolicy, target.Overrides)

				// Parse the backup time string ("15:04") into a time.Time once
				// here so all downstream consumers get a ready-to-use value.
				if merged.Spec.Schedule.Full != nil {
					t, err := parseBackupTime(merged.Spec.Schedule.Full.Time)
					if err != nil {
						errs = append(errs, fmt.Errorf(
							"assignment %q target %s/%s: invalid schedule.full.time %q: %w",
							a.Metadata.Name, target.ResourceGroup, target.ResourceName,
							merged.Spec.Schedule.Full.Time, err,
						))
						continue
					}
					merged.Spec.Schedule.Full.ParsedTime = t
				}

				rt := models.ResolvedTarget{
					SubscriptionID: a.Metadata.SubscriptionID,
					PolicyName:     pa.Policy,
					Target:         target,
					Policy:         merged,
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
// RetentionSpec is a flat struct so we merge non-zero fields individually.
// ScheduleSpec uses pointers so nil means "not overridden".
func applyOverrides(base models.Policy, overrides *models.PolicyOverrides) models.Policy {
	if overrides == nil {
		return base
	}
	merged := base

	if overrides.Retention != nil {
		r := *overrides.Retention
		if r.Daily != nil && r.Daily.Count > 0 {
			merged.Spec.Retention.Daily.Count = r.Daily.Count
		}
		if r.Weekly != nil && r.Weekly.Count > 0 {
			merged.Spec.Retention.Weekly.Count = r.Weekly.Count
		}
		if r.Weekly != nil && r.Weekly.Day != "" {
			merged.Spec.Retention.Weekly.Day = r.Weekly.Day
		}
		if r.Monthly != nil && r.Monthly.Count > 0 {
			merged.Spec.Retention.Monthly.Count = r.Monthly.Count
		}
		if r.Yearly != nil && r.Yearly.Count > 0 {
			merged.Spec.Retention.Yearly.Count = r.Yearly.Count
		}
		if r.Yearly != nil && r.Yearly.Month != "" {
			merged.Spec.Retention.Yearly.Month = r.Yearly.Month
		}
		if r.Yearly != nil && r.Yearly.Day != "" {
			merged.Spec.Retention.Yearly.Day = r.Yearly.Day
		}
		if r.Yearly != nil && r.Yearly.Week != "" {
			merged.Spec.Retention.Yearly.Week = r.Yearly.Week
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

// parseBackupTime parses a "15:04" UTC time string into a time.Time.
// The date component is set to the zero date; only the time matters to Azure.
func parseBackupTime(s string) (time.Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected HH:MM format, got %q", s)
	}
	return t.UTC(), nil
}
