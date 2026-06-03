package mdoel_azure

// internal/models/providers/azure/azure.go

// AzureResource is the common Azure resource block — Layer 2.
// Shared across all Azure resource kinds (vm, disk, sqlvm).
// The ResourceKind field routes to the correct Layer 3 spec.
type AzureResource struct {
	SubscriptionID string `yaml:"subscription_id" validate:"required"`
	ResourceKind   string `yaml:"resource_kind"   validate:"required,oneof=vm disk sqlvm sqlpaas"`
	ResourceGroup  string `yaml:"resource_group"  validate:"required"`
	VaultName      string `yaml:"vault_name"      validate:"required,min=2,max=50"`
}
