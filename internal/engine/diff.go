package engine

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// DiffPolicy compares a local BackupPolicy against the live Azure policy.
// Returns a slice of human-readable difference strings.
func DiffPolicy(local *models.BackupPolicy, remote *armrecoveryservicesbackup.ProtectionPolicyResource) []string {
	var diffs []string

	// Example: compare the policy ID/name
	if remote.Name != nil && *remote.Name != local.Name {
		diffs = append(diffs, fmt.Sprintf("name: local=%q, azure=%q", local.Name, *remote.Name))
	}

	// Add more field comparisons here as your BackupPolicy struct grows

	return diffs
}
