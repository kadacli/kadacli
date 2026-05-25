package audit

import (
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// FindUnprotected returns resources from the discovered set that have no matching
// assignment in the resolved targets.
// TODO: implement for 1.0.0
func FindUnprotected(discovered []models.Resource, resolved []models.ResolvedTarget) []models.Resource {
	assignedKeys := make(map[string]struct{}, len(resolved))
	for _, rt := range resolved {
		key := rt.Target.ResourceGroup + "/" + rt.Target.ResourceName
		assignedKeys[key] = struct{}{}
	}

	var unprotected []models.Resource
	for _, r := range discovered {
		key := r.ResourceGroup + "/" + r.Name
		if _, covered := assignedKeys[key]; !covered {
			unprotected = append(unprotected, r)
		}
	}
	return unprotected
}
