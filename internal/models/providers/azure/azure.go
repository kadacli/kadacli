// internal/models/providers/azure/azure.go
package azure

// AzureResource is the common Azure resource block — Layer 2.
// Shared across all Azure resource kinds (vm, disk, sqlvm).
// The ResourceKind field routes to the correct Layer 3 spec.
type AzureResource struct {
	SubscriptionID string `yaml:"subscriptionId" validate:"required"`
	ResourceKind   string `yaml:"resourceKind"   validate:"required,oneof=vm disk sqlvm sqlpaas"`
	ResourceGroup  string `yaml:"resourceGroup"  validate:"required"`
	VaultName      string `yaml:"vaultName"      validate:"required,min=2,max=50"`
}
