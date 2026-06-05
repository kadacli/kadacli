package azure_engine

// internal/engine/azure_engine/parser.go

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	azure_models "github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models"
	azure_assignments "github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models/azure_assignments"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models/azure_policies"
)

// ParsePolicy decodes and validates the Azure-specific resource and spec blocks
// of a policy. Returns a slice of errors so all problems are reported at once.
func ParsePolicy(policy models.Policy) []error {
	var errs []error

	// Decode and validate the resource block
	var resource azure_models.AzureResource
	if err := policy.Resource.Decode(&resource); err != nil {
		return []error{fmt.Errorf("decoding resource block: %w", err)}
	}
	if resource.SubscriptionID == "" {
		errs = append(errs, fmt.Errorf("resource.subscription_id is required"))
	}
	if resource.ResourceKind == "" {
		errs = append(errs, fmt.Errorf("resource.resource_kind is required"))
	}
	if resource.ResourceGroup == "" {
		errs = append(errs, fmt.Errorf("resource.resource_group is required"))
	}
	if resource.VaultName == "" {
		errs = append(errs, fmt.Errorf("resource.vault_name is required"))
	}

	// Decode and validate the spec block — route by resource kind
	switch resource.ResourceKind {
	case "vm":
		errs = append(errs, parseVMSpec(policy)...)
	default:
		errs = append(errs, fmt.Errorf("resource_kind %q is not supported", resource.ResourceKind))
	}

	return errs
}

// parseVMSpec decodes and validates the VM-specific spec block.
func parseVMSpec(policy models.Policy) []error {
	var errs []error

	var spec azure_policies.AzureVMSpec
	if err := policy.Spec.Decode(&spec); err != nil {
		return []error{fmt.Errorf("decoding spec block: %w", err)}
	}

	if spec.InstantRPRetentionDays == 0 {
		errs = append(errs, fmt.Errorf("spec.instant_rp_retention_days is required"))
	}
	if spec.Retention.Daily == nil {
		errs = append(errs, fmt.Errorf("spec.retention.daily is required"))
	}
	if spec.Schedule.Full == nil {
		errs = append(errs, fmt.Errorf("spec.schedule.full is required"))
	} else {
		if spec.Schedule.Full.Frequency == "" {
			errs = append(errs, fmt.Errorf("spec.schedule.full.frequency is required"))
		}
		if spec.Schedule.Full.Time == "" {
			errs = append(errs, fmt.Errorf("spec.schedule.full.time is required"))
		}
	}

	return errs
}

// ParseAssignment decodes and validates the Azure-specific resource and
// assignments blocks. Returns a slice of errors so all problems are reported at once.
func ParseAssignment(assignment models.Assignment) []error {
	var errs []error

	// Decode and validate the resource block
	var resource azure_models.AzureResource
	if err := assignment.Resource.Decode(&resource); err != nil {
		return []error{fmt.Errorf("decoding resource block: %w", err)}
	}
	if resource.SubscriptionID == "" {
		errs = append(errs, fmt.Errorf("resource.subscription_id is required"))
	}
	if resource.ResourceGroup == "" {
		errs = append(errs, fmt.Errorf("resource.resource_group is required"))
	}
	if resource.VaultName == "" {
		errs = append(errs, fmt.Errorf("resource.vault_name is required"))
	}

	// Decode and validate the assignments block
	var vmConfig azure_assignments.VMConfig
	if err := assignment.Assignments.Decode(&vmConfig); err != nil {
		return []error{fmt.Errorf("decoding assignments block: %w", err)}
	}
	if len(vmConfig.Assignments) == 0 {
		errs = append(errs, fmt.Errorf("assignments must not be empty"))
	}
	for i, a := range vmConfig.Assignments {
		if a.Policy == "" {
			errs = append(errs, fmt.Errorf("assignments[%d].policy is required", i))
		}
		if len(a.Targets) == 0 {
			errs = append(errs, fmt.Errorf("assignments[%d].targets must not be empty", i))
		}
		for j, t := range a.Targets {
			if t.ResourceGroup == "" {
				errs = append(errs, fmt.Errorf("assignments[%d].targets[%d].resource_group is required", i, j))
			}
			if t.ResourceName == "" {
				errs = append(errs, fmt.Errorf("assignments[%d].targets[%d].resource_name is required", i, j))
			}
		}
	}

	return errs
}
