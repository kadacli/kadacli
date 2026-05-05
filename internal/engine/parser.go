package engine

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// ParsePolicy validates top-level structural requirements of a Policy
// before it enters the resolver. Per-resource validation is in validator.go.
func ParsePolicy(p models.Policy) []error {
	var errs []error

	if p.APIVersion == "" {
		errs = append(errs, fmt.Errorf("apiVersion is required"))
	} else if p.APIVersion != "backupctl/v1" {
		errs = append(errs, fmt.Errorf("apiVersion must be backupctl/v1, got %q", p.APIVersion))
	}
	if p.Kind != "Policy" {
		errs = append(errs, fmt.Errorf("kind must be Policy, got %q", p.Kind))
	}
	if p.Metadata.Name == "" {
		errs = append(errs, fmt.Errorf("metadata.name is required"))
	}
	if p.Metadata.Owner == "" {
		errs = append(errs, fmt.Errorf("metadata.owner is required"))
	}
	if p.Spec.Timezone == "" {
		errs = append(errs, fmt.Errorf("spec.timezone is required"))
	}
	if p.Spec.Vault.Name == "" {
		errs = append(errs, fmt.Errorf("spec.vault.name is required"))
	}
	if p.Spec.Vault.ResourceGroup == "" {
		errs = append(errs, fmt.Errorf("spec.vault.resource_group is required"))
	}

	return errs
}

// ParseAssignment validates top-level structural requirements of an Assignment.
func ParseAssignment(a models.Assignment) []error {
	var errs []error

	if a.APIVersion == "" {
		errs = append(errs, fmt.Errorf("apiVersion is required"))
	} else if a.APIVersion != "backupctl/v1" {
		errs = append(errs, fmt.Errorf("apiVersion must be backupctl/v1, got %q", a.APIVersion))
	}
	if a.Kind != "Assignment" {
		errs = append(errs, fmt.Errorf("kind must be Assignment, got %q", a.Kind))
	}
	if a.Metadata.Name == "" {
		errs = append(errs, fmt.Errorf("metadata.name is required"))
	}
	if a.Metadata.SubscriptionID == "" {
		errs = append(errs, fmt.Errorf("metadata.subscriptionId is required"))
	}
	if a.Metadata.Owner == "" {
		errs = append(errs, fmt.Errorf("metadata.owner is required"))
	}
	if len(a.Spec.Assignments) == 0 {
		errs = append(errs, fmt.Errorf("spec.assignments must not be empty"))
	}
	for i, pa := range a.Spec.Assignments {
		if pa.Policy == "" {
			errs = append(errs, fmt.Errorf("spec.assignments[%d].policy is required", i))
		}
		if len(pa.Targets) == 0 {
			errs = append(errs, fmt.Errorf("spec.assignments[%d].targets must not be empty", i))
		}
		for j, t := range pa.Targets {
			if t.Type == "" {
				errs = append(errs, fmt.Errorf("spec.assignments[%d].targets[%d].type is required", i, j))
			}
			if t.ResourceGroup == "" {
				errs = append(errs, fmt.Errorf("spec.assignments[%d].targets[%d].resource_group is required", i, j))
			}
			if t.ResourceName == "" {
				errs = append(errs, fmt.Errorf("spec.assignments[%d].targets[%d].resource_name is required", i, j))
			}
		}
	}

	return errs
}
