package azure_models

// internal/models/azure_models/resolved.go

import (
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models/azure_policies"
)

// ResolvedVM is the internal representation of a single VM backup target,
// with all policy data merged and ready to pass to external/azure.
type ResolvedVM struct {
	SubscriptionID     string
	ResourceGroup      string
	ResourceName       string
	VaultName          string
	IncludeFutureDisks bool
	PolicyName         string
	Resource           AzureResource
	Spec               azure_policies.AzureVMSpec
}
