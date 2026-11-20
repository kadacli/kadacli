package azure

import (
	"fmt"

	"github.com/kadacli/kadacli/internal/models/azure/assignments"
	"github.com/kadacli/kadacli/internal/models/azure/policies"
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
	Timezone       string // from Policy.Metadata.Timezone
	DiskSelection  *assignments.DiskSelection
	Resource       Resource
	Spec           policies.AzureVMSpec
}

type ResolvedDisk struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceName   string
	VaultName      string // convenience alias for Resource.Vault.Name
	PolicyName     string
	Timezone       string // from Policy.Metadata.Timezone
	Resource       Resource
	Spec           policies.AzureDiskSpec
}

// SnapshotResourceGroupID is the ARM ID of the resource group snapshots are
// written to: spec.snapshot_resource_group when set, otherwise the disk's own
// resource group. Both the apply path and the differ read it from here so the
// value compared is always the value sent.
func (d ResolvedDisk) SnapshotResourceGroupID() string {
	rg := d.ResourceGroup
	if d.Spec.SnapshotResourceGroup != "" {
		rg = d.Spec.SnapshotResourceGroup
	}
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", d.SubscriptionID, rg)
}

type ResolvedSqlDb struct {
	SubscriptionID string
	ResourceGroup  string
	ServerName     string
	ResourceName   string
	VaultName      string
	PolicyName     string
	Timezone       string
	Resource       Resource
	Spec           policies.AzureSqlDbSpec
	Overrides      *assignments.SqlDbOverrides
}
