package azure_external

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
)

// GetBackupPolicySQL retrieves the AzureVMWorkloadProtectionPolicy for an Azure SQL Database target.
func (c *Client) GetBackupPolicySQL(
	ctx context.Context,
	resourceGroup, vaultName, policyName string,
) (*armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy, error) {
	client, err := armrecoveryservicesbackup.NewProtectionPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create protection policies client: %w", err)
	}

	resp, err := client.Get(ctx, vaultName, resourceGroup, policyName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup policy %q from vault %q: %w", policyName, vaultName, err)
	}

	sqlPolicy, ok := resp.Properties.(*armrecoveryservicesbackup.AzureVMWorkloadProtectionPolicy)
	if !ok {
		return nil, fmt.Errorf("policy %q is not an AzureVMWorkloadProtectionPolicy (got %T)", policyName, resp.Properties)
	}
	return sqlPolicy, nil
}
