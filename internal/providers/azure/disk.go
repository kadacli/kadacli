package azure

import (
	"context"
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// ApplyDisk applies the backup policy to an Azure Managed Disk target.
// Uses Backup Vault (Microsoft.DataProtection/backupVaults) — a different
// Azure API namespace to the Recovery Services Vault used by other resource types.
// TODO: implement for 1.0.0; requires github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dataprotection/armdataprotection
func (c *Client) ApplyDisk(ctx context.Context, rt models.ResolvedTarget) error {
	return fmt.Errorf("azure_disk: ApplyDisk not yet implemented")
}

// GetBackupPolicyDisk retrieves the current backup configuration for an Azure Managed Disk.
// TODO: implement for 1.0.0
func (c *Client) GetBackupPolicyDisk(
	ctx context.Context,
	resourceGroup, vaultName, diskName string,
) (interface{}, error) {
	return nil, fmt.Errorf("azure_disk: GetBackupPolicyDisk not yet implemented")
}
