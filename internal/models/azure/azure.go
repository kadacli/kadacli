package azure

// AzureResource is the common Azure resource block — Layer 2.
// Shared across all Azure resource kinds (vm, disk, sqlvm).
// The Type field routes to the correct Layer 3 spec.
type AzureResource struct {
	SubscriptionID string     `yaml:"subscription_id" validate:"required"`
	Type           string     `yaml:"type"            validate:"required,oneof=azure_vm azure_disk azure_sql_vm azure_sql_db"`
	Vault          AzureVault `yaml:"vault"           validate:"required"`
}

// AzureVault identifies the Recovery Services Vault that owns this policy.
type AzureVault struct {
	Name          string `yaml:"name"           validate:"required,min=2,max=50"`
	ResourceGroup string `yaml:"resource_group" validate:"required"`
}
