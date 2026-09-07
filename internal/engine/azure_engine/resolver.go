// internal/engine/azure_engine/resolver.go
package azure_engine

import (
	"fmt"
	"time"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
	azassignments "github.com/dbjorklund/backup-policy-as-code/internal/models/azure/assignments"
	azpolicies "github.com/dbjorklund/backup-policy-as-code/internal/models/azure/policies"
)

// ResolveVMs matches each assignment target to its named policy and returns
// a flat slice of ResolvedVMs with all data merged and validated.
func ResolveVMs(policies []models.Policy, assignments []models.Assignment) ([]azure.ResolvedVM, []error) {
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
			errs = append(errs, fmt.Errorf("Duplicate policy name %q", p.Metadata.Name))
			continue
		}
		policyIndex[p.Metadata.Name] = p
	}

	// Validate assignments
	for _, a := range assignments {
		if assignErrs := ParseAssignment(a); len(assignErrs) > 0 {
			for _, e := range assignErrs {
				errs = append(errs, fmt.Errorf("Assignment %q: %w", a.Metadata.Name, e))
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Resolve each assignment target against its policy
	var resolved []azure.ResolvedVM
	for _, a := range assignments {
		// Decode the assignments block
		var vmAssignments []azassignments.VMAssignment
		if err := a.Assignments.Decode(&vmAssignments); err != nil {
			errs = append(errs, fmt.Errorf("Assignment %q: decoding assignments block: %w", a.Metadata.Name, err))
			continue
		}

		for _, pa := range vmAssignments {
			policy, ok := policyIndex[pa.Policy]
			if !ok {
				errs = append(errs, fmt.Errorf(
					"Assignment %q references unknown policy %q", a.Metadata.Name, pa.Policy,
				))
				continue
			}

			// Decode the policy resource block — vault and subscription live here
			var policyResource azure.Resource
			if err := policy.Resource.Decode(&policyResource); err != nil {
				errs = append(errs, fmt.Errorf(
					"Assignment %q policy %q: decoding resource: %w", a.Metadata.Name, pa.Policy, err,
				))
				continue
			}

			// Decode the policy spec
			var spec azpolicies.AzureVMSpec
			if err := policy.Spec.Decode(&spec); err != nil {
				errs = append(errs, fmt.Errorf(
					"Assignment %q policy %q: decoding spec: %w", a.Metadata.Name, pa.Policy, err,
				))
				continue
			}

			// Parse the backup time once here so downstream consumers get a
			// ready-to-use value
			if spec.Schedule.Full != nil {
				t, err := parseBackupTime(spec.Schedule.Full.Time)
				if err != nil {
					errs = append(errs, fmt.Errorf(
						"Assignment %q policy %q: invalid schedule.full.time %q: %w",
						a.Metadata.Name, pa.Policy, spec.Schedule.Full.Time, err,
					))
					continue
				}
				spec.Schedule.Full.ParsedTime = t
			}

			for _, target := range pa.Targets {
				resolved = append(resolved, azure.ResolvedVM{
					SubscriptionID: policyResource.SubscriptionID,
					ResourceGroup:  target.ResourceGroup,
					ResourceName:   target.ResourceName,
					VaultName:      policyResource.Vault.Name,
					PolicyName:     pa.Policy,
					Timezone:       policy.Metadata.Timezone,
					DiskSelection:  target.DiskSelection,
					Resource:       policyResource,
					Spec:           spec,
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
		return time.Time{}, fmt.Errorf("Expected HH:MM format, got %q", s)
	}
	return t.UTC(), nil
}

// ResolveDisks matches each assignment target to its named policy and returns
// a flat slice of ResolvedDisks with all data merged and validated.
func ResolveDisks(policies []models.Policy, assignments []models.Assignment) ([]azure.ResolvedDisk, []error) {
	var errs []error

	// Validate and index policies by name
	policyIndex := make(map[string]models.Policy, len(policies))
	for _, p := range policies {
		if parseErrs := ParsePolicy(p); len(parseErrs) > 0 {
			for _, e := range parseErrs {
				errs = append(errs, fmt.Errorf("Policy %q: %w", p.Metadata.Name, e))
			}
			continue
		}
		if _, exists := policyIndex[p.Metadata.Name]; exists {
			errs = append(errs, fmt.Errorf("Duplicate policy name %q", p.Metadata.Name))
			continue
		}
		policyIndex[p.Metadata.Name] = p
	}

	// Validate assignments
	for _, a := range assignments {

		if assignErrs := ParseAssignment(a); len(assignErrs) > 0 {
			for _, e := range assignErrs {
				errs = append(errs, fmt.Errorf("Assignment %q: %w", a.Metadata.Name, e))
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Resolve each assignment target against its policy
	var resolved []azure.ResolvedDisk
	for _, a := range assignments {
		// Decode the assignments block
		// resolveAzure has already routed this assignment here by resource type,
		// so a decode failure is a malformed file, not a different provider's
		// assignment. Skipping it silently drops every target it declares and
		// still reports success.
		var diskAssignments []azassignments.DiskAssignment
		if err := a.Assignments.Decode(&diskAssignments); err != nil {
			errs = append(errs, fmt.Errorf("Assignment %q: decoding assignments block: %w", a.Metadata.Name, err))
			continue
		}

		for _, pa := range diskAssignments {
			policy, ok := policyIndex[pa.Policy]
			if !ok {
				errs = append(errs, fmt.Errorf(
					"Assignment %q references unknown policy %q", a.Metadata.Name, pa.Policy,
				))
				continue
			}

			// Decode the policy resource block
			var policyResource azure.Resource
			if err := policy.Resource.Decode(&policyResource); err != nil {
				errs = append(errs, fmt.Errorf(
					"Assignment %q policy %q: decoding resource: %w", a.Metadata.Name, pa.Policy, err,
				))
				continue
			}

			// Decode the policy spec
			var spec azpolicies.AzureDiskSpec
			if err := policy.Spec.Decode(&spec); err != nil {
				errs = append(errs, fmt.Errorf(
					"Assignment %q policy %q: decoding spec: %w", a.Metadata.Name, pa.Policy, err,
				))
				continue
			}

			// Azure interprets a schedule with no time zone as UTC. Carry the
			// policy's declared zone through so the backup runs when the YAML
			// says it does; fall back to UTC explicitly rather than sending an
			// empty value.
			timezone := policy.Metadata.Timezone
			if timezone == "" {
				timezone = "UTC"
			}

			for _, target := range pa.Targets {
				// DiskTarget.Overrides decodes but ResolvedDisk has nowhere to
				// put it, so nothing downstream could honour it. Reject the
				// intent rather than dropping it silently — same call as for
				// azure_sql_db.
				if target.Overrides != nil {
					errs = append(errs, fmt.Errorf(
						"Assignment %q target %s/%s: per-target overrides are not supported for azure_disk — remove the 'overrides' block and set the value on the policy instead",
						a.Metadata.Name, target.ResourceGroup, target.ResourceName,
					))
					continue
				}

				resolved = append(resolved, azure.ResolvedDisk{
					SubscriptionID: policyResource.SubscriptionID,
					ResourceGroup:  target.ResourceGroup,
					ResourceName:   target.ResourceName,
					VaultName:      policyResource.Vault.Name,
					PolicyName:     pa.Policy,
					Timezone:       timezone,
					Resource:       policyResource,
					Spec:           spec,
				})
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return resolved, nil
}

func ResolveSQLDBs(policies []models.Policy, assignments []models.Assignment) ([]azure.ResolvedSqlDb, []error) {
	var errs []error

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

	var resolved []azure.ResolvedSqlDb
	for _, a := range assignments {
		// Same as ResolveDisks: routing already established this is a SQL DB
		// assignment, so a decode failure means the file is malformed.
		var sqlDbAssignments []azassignments.SqlDbAssignment
		if err := a.Assignments.Decode(&sqlDbAssignments); err != nil {
			errs = append(errs, fmt.Errorf("Assignment %q: decoding assignments block: %w", a.Metadata.Name, err))
			continue
		}

		for _, pa := range sqlDbAssignments {
			policy, ok := policyIndex[pa.Policy]
			if !ok {
				errs = append(errs, fmt.Errorf(
					"Assignment %q references unknown policy %q", a.Metadata.Name, pa.Policy,
				))
				continue
			}

			var policyResource azure.Resource
			if err := policy.Resource.Decode(&policyResource); err != nil {
				errs = append(errs, fmt.Errorf(
					"Assignment %q policy %q: decoding resource: %w", a.Metadata.Name, pa.Policy, err,
				))
				continue
			}

			var spec azpolicies.AzureSqlDbSpec
			if err := policy.Spec.Decode(&spec); err != nil {
				errs = append(errs, fmt.Errorf(
					"Assignment %q policy %q: decoding spec: %w", a.Metadata.Name, pa.Policy, err,
				))
				continue
			}

			// Apply the documented default: differential_backup_frequency
			// defaults to 24h when omitted (cmd/templates/policy/azure_sql_db.yml).
			// Azure only accepts 12 or 24, so a zero value (field omitted) must
			// never reach ApplySqlDbSTR — it is rejected as
			// InvalidDiffBackupIntervalHours. The validate-time tag is
			// omitempty,oneof=12 24, so 0 passes validation and would otherwise
			// fail only at apply.
			if spec.ShortTermRetention.DifferentialBackupFrequency == 0 {
				spec.ShortTermRetention.DifferentialBackupFrequency = 24
			}

			for _, target := range pa.Targets {
				// ParseAssignment validates every assignment against the VM
				// target shape, which has no server_name, so nothing upstream
				// has checked this one. An Azure SQL database is addressed as
				// resource_group/server_name/database — without the server the
				// request is malformed and only fails once it reaches Azure.
				if target.ServerName == "" {
					errs = append(errs, fmt.Errorf(
						"Assignment %q target %q: server_name is required for azure_sql_db targets",
						a.Metadata.Name, target.ResourceName,
					))
					continue
				}

				// Per-target overrides are not yet honored by DiffSqlDb/ApplySqlDb
				// (SqlDbTarget.Overrides is parsed but never read). Rather than
				// silently ignoring a user's declared override, reject it at
				// validate time so the intent fails loud instead of being dropped.
				if target.Overrides != nil {
					errs = append(errs, fmt.Errorf(
						"Assignment %q target %s/%s: per-target overrides are not supported for azure_sql_db — remove the 'overrides' block and set the value on the policy instead",
						a.Metadata.Name, target.ServerName, target.ResourceName,
					))
					continue
				}
				resolved = append(resolved, azure.ResolvedSqlDb{
					SubscriptionID: policyResource.SubscriptionID,
					ResourceGroup:  target.ResourceGroup,
					ServerName:     target.ServerName,
					ResourceName:   target.ResourceName,
					VaultName:      policyResource.Vault.Name,
					PolicyName:     pa.Policy,
					Timezone:       policy.Metadata.Timezone,
					Resource:       policyResource,
					Spec:           spec,
					Overrides:      target.Overrides,
				})
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return resolved, nil
}
