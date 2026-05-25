package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
)

// GetVault retrieves a Recovery Services Vault. Used by azure_vm, azure_sql, and azure_sql_vm targets.
func (c *Client) GetVault(ctx context.Context, resourceGroup, vaultName string) (*armrecoveryservices.Vault, error) {
	client, err := armrecoveryservices.NewVaultsClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vaults client: %w", err)
	}
	result, err := client.Get(ctx, resourceGroup, vaultName, nil)
	if err != nil {
		return nil, fmt.Errorf("vault %q not found in resource group %q: %w", vaultName, resourceGroup, err)
	}
	return &result.Vault, nil
}

// GetBackupPolicyID retrieves the ARM resource ID of a named backup policy from a vault.
func (c *Client) GetBackupPolicyID(ctx context.Context, resourceGroup, vaultName, policyName string) (string, error) {
	client, err := armrecoveryservicesbackup.NewProtectionPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create protection policies client: %w", err)
	}
	result, err := client.Get(ctx, vaultName, resourceGroup, policyName, nil)
	if err != nil {
		return "", fmt.Errorf("policy %q not found in vault %q: %w", policyName, vaultName, err)
	}
	if result.ID == nil {
		return "", fmt.Errorf("policy %q exists but has no ID", policyName)
	}
	return *result.ID, nil
}

// VaultTypeForTarget returns the Azure vault namespace for a given target type.
// azure_disk uses Backup Vault (Microsoft.DataProtection); all others use Recovery Services Vault.
func VaultTypeForTarget(targetType string) string {
	if targetType == "azure_disk" {
		return "Microsoft.DataProtection/backupVaults"
	}
	return "Microsoft.RecoveryServices/vaults"
}
