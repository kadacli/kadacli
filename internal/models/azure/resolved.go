package azure

import (
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure/assignments"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure/policies"
)

// ResolvedVM is the internal representation of a single VM backup target,
// with all policy data merged and ready to pass to external/azure.
// VaultName is a convenience alias for Resource.Vault.Name.
type ResolvedVM struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceName   string
	VaultName      string // convenience alias for Resource.Vault.Name
	PolicyName     string
	DiskSelection  *assignments.DiskSelection
	Resource       AzureResource
	Spec           policies.AzureVMSpec
}
