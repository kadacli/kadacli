package azure_engine

// internal/engine/azure_engine/parser.go

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure/assignments"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure/policies"
	"github.com/go-playground/validator/v10"
)

// ParsePolicy decodes and validates the Azure-specific resource and spec blocks
// of a policy. Returns a slice of errors so all problems are reported at once.
func ParsePolicy(policy models.Policy) []error {

	var errs []error

	// Decode and validate the resource block
	var resource azure.AzureResource
	if err := policy.Resource.Decode(&resource); err != nil {
		return []error{fmt.Errorf("Decoding resource block: %w", err)}
	}
	if resource.SubscriptionID == "" {
		errs = append(errs, fmt.Errorf("resource.subscription_id is required"))
	}
	if resource.Type == "" {
		errs = append(errs, fmt.Errorf("resource.type is required"))
	}
	if resource.Vault.Name == "" {
		errs = append(errs, fmt.Errorf("resource.vault.name is required"))
	}
	if resource.Vault.ResourceGroup == "" {
		errs = append(errs, fmt.Errorf("resource.vault.resource_group is required"))
	}

	// Decode and validate the spec block — route by resource type
	switch resource.Type {
	case "azure_vm":
		errs = append(errs, parseVMSpec(policy)...)
	case "azure_disk":
		errs = append(errs, parseDiskSpec(policy)...)
	case "azure_sql_db":
		errs = append(errs, parseSqlDbSpec(policy)...)
	default:
		errs = append(errs, fmt.Errorf("resource.type %q is not supported", resource.Type))
	}

	return errs
}

func parseSqlDbSpec(policy models.Policy) []error {
	var errs []error
	var spec policies.AzureSqlDbSpec

	validate := validator.New()

	if err := policy.Spec.Decode(&spec); err != nil {
		return []error{fmt.Errorf("Decoding spec block: %w", err)}
	}

	if err := policy.Spec.Decode(&spec); err != nil {
		return []error{fmt.Errorf("Decoding spec block: %w", err)}
	}

	if err := validate.Struct(spec); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			errs = append(errs, fmt.Errorf("Field '%s' failed validation: %s, parameter: %s", e.Field(), e.Tag(), e.Param()))
		}
	}
	return errs
}

func parseDiskSpec(policy models.Policy) []error {
	var errs []error
	var spec policies.AzureDiskSpec

	validate := validator.New()

	if err := policy.Spec.Decode(&spec); err != nil {
		return []error{fmt.Errorf("Decoding spec block: %w", err)}
	}

	if !spec.Retention.Default && len(spec.Retention.RetentionRules) == 0 {
		errs = append(errs, fmt.Errorf("spec.retention.retentienRules cannot be empty if spec.retention.default is false"))
	}

	err := validate.Struct(spec.Retention)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			errs = append(errs, fmt.Errorf("Field '%s' failed validation: %s, parameter: %s", e.Field(), e.Tag(), e.Param()))
		}
	}

	err = validate.Struct(spec.Schedule)
	if err != nil {
		errs = append(errs, err)
	}

	if spec.Schedule.Frequency == "" {
		errs = append(errs, fmt.Errorf("spec.schedule.frequency is required"))
	}

	return errs

}

// parseVMSpec decodes and validates the VM-specific spec block.
func parseVMSpec(policy models.Policy) []error {
	var errs []error

	var spec policies.AzureVMSpec
	if err := policy.Spec.Decode(&spec); err != nil {
		return []error{fmt.Errorf("Decoding spec block: %w", err)}
	}

	if spec.InstantRPRetentionDays == 0 {
		errs = append(errs, fmt.Errorf("spec.instant_rp_retention_days is required"))
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

	// Decode and validate the assignments block
	var vmAssignments []assignments.VMAssignment
	if err := assignment.Assignments.Decode(&vmAssignments); err != nil {
		return []error{fmt.Errorf("Decoding assignments block: %w", err)}
	}
	if len(vmAssignments) == 0 {
		errs = append(errs, fmt.Errorf("Assignments must not be empty"))
	}
	for i, a := range vmAssignments {
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
			if ds := t.DiskSelection; ds != nil {
				if (ds.Mode == "include" || ds.Mode == "exclude") && len(ds.LUNs) == 0 {
					errs = append(errs, fmt.Errorf("assignments[%d].targets[%d].disk_selection.luns must not be empty when mode is %q", i, j, ds.Mode))
				}
			}
		}
	}

	return errs
}
