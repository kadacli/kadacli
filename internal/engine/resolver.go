package engine

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

// Resolve merges all validated assignments against their policies and returns
// a flat slice of ResolvedTargets ready for diff and apply.
// Policies and assignments from different providers are handled independently.
func Resolve(pathPolicies []models.PathPolicy, pathAssignments []models.PathAssignment) ([]models.ResolvedTarget, []error) {
	var errs []error
	var resolved []models.ResolvedTarget

	// Split into provider buckets — only allocate slices for providers that appear
	policiesByProvider := make(map[string][]models.Policy)
	assignmentsByProvider := make(map[string][]models.Assignment)

	for _, pp := range pathPolicies {
		switch pp.Policy.Provider {
		case "azure", "aws": // extend as providers are added
			policiesByProvider[pp.Policy.Provider] = append(policiesByProvider[pp.Policy.Provider], pp.Policy)
		default:
			errs = append(errs, fmt.Errorf("resolve: provider %q is not yet supported", pp.Policy.Provider))
		}
	}

	for _, pa := range pathAssignments {
		switch pa.Assignment.Provider {
		case "azure", "aws":
			assignmentsByProvider[pa.Assignment.Provider] = append(assignmentsByProvider[pa.Assignment.Provider], pa.Assignment)
		default:
			errs = append(errs, fmt.Errorf("resolve: provider %q is not yet supported", pa.Assignment.Provider))
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Azure
	if len(policiesByProvider["azure"]) > 0 || len(assignmentsByProvider["azure"]) > 0 {
		targets, resolveErrs := resolveAzure(policiesByProvider["azure"], assignmentsByProvider["azure"])
		if len(resolveErrs) > 0 {
			errs = append(errs, resolveErrs...)
		} else {
			resolved = append(resolved, targets...)
		}
	}

	// AWS (future)
	// if len(policiesByProvider["aws"]) > 0 || len(assignmentsByProvider["aws"]) > 0 {
	// 	targets, resolveErrs := resolveAWS(policiesByProvider["aws"], assignmentsByProvider["aws"])
	// 	...
	// }

	if len(errs) > 0 {
		return nil, errs
	}
	fmt.Printf("DEBUG: length of resolved: %d\n", len(resolved))
	return resolved, nil
}

// resolveAzure handles all Azure-specific resolution logic, splitting by resource type.
func resolveAzure(policies []models.Policy, assignments []models.Assignment) ([]models.ResolvedTarget, []error) {
	var errs []error
	var resolved []models.ResolvedTarget

	// Index policies by name so we can look up resource type
	policyIndex := make(map[string]models.Policy, len(policies))
	for _, p := range policies {
		policyIndex[p.Metadata.Name] = p
	}

	var vmAssignments []models.Assignment
	var diskAssignments []models.Assignment

	for _, a := range assignments {
		var assignmentList []struct {
			Policy string `yaml:"policy"`
		}
		if err := a.Assignments.Decode(&assignmentList); err != nil || len(assignmentList) == 0 {
			errs = append(errs, fmt.Errorf("assignment %q: decoding policy reference: %w", a.Metadata.Name, err))
			continue
		}

		// Use the first policy reference to determine resource type
		p, ok := policyIndex[assignmentList[0].Policy]
		if !ok {
			errs = append(errs, fmt.Errorf("assignment %q: unknown policy %q", a.Metadata.Name, assignmentList[0].Policy))
			continue
		}

		var resource azure.AzureResource
		if err := p.Resource.Decode(&resource); err != nil {
			errs = append(errs, fmt.Errorf("assignment %q: decoding resource: %w", a.Metadata.Name, err))
			continue
		}

		fmt.Printf("DEBUG: Resource.Type: %s\n", resource.Type)

		switch resource.Type {
		case "azure_disk":
			diskAssignments = append(diskAssignments, a)
		case "azure_vm":
			vmAssignments = append(vmAssignments, a)
		default:
			errs = append(errs, fmt.Errorf("Assignment type not supports for assignment: %s", a.Metadata.Name))
		}
	}

	
	vms, resolveErrs := azure_engine.ResolveVMs(policies, vmAssignments)
	if len(resolveErrs) > 0 {
		errs = append(errs, resolveErrs...)
	} else {
		for _, vm := range vms {
			resolved = append(resolved, models.ResolvedTarget{Provider: "azure", Payload: vm})
		}
	}

	disks, resolveErrs := azure_engine.ResolveDisks(policies, diskAssignments)
	if len(resolveErrs) > 0 {
		errs = append(errs, resolveErrs...)
	} else {
		for _, disk := range disks {
			resolved = append(resolved, models.ResolvedTarget{Provider: "azure", Payload: disk})
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return resolved, nil
}
