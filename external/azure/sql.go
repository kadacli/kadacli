package azure_external

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

// GetSqlDb checks whether an Azure SQL Database exists.
// Returns (nil, nil) if the database does not exist (HTTP 404).
func (c *Client) GetSqlDb(
	ctx context.Context,
	resourceGroup, serverName, databaseName string,
) (*armsql.Database, error) {
	client, err := armsql.NewDatabasesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create databases client: %w", err)
	}

	resp, err := client.Get(ctx, resourceGroup, serverName, databaseName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get SQL database %q on server %q: %w", databaseName, serverName, err)
	}

	return &resp.Database, nil
}

// GetSqlDbSTR retrieves the short term retention policy for an Azure SQL Database.
func (c *Client) GetSqlDbSTR(
	ctx context.Context,
	resourceGroup, serverName, databaseName string,
) (*armsql.BackupShortTermRetentionPolicy, error) {
	client, err := armsql.NewBackupShortTermRetentionPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create STR client: %w", err)
	}

	resp, err := client.Get(ctx, resourceGroup, serverName, databaseName, armsql.ShortTermRetentionPolicyNameDefault, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get STR policy for db %q on server %q: %w", databaseName, serverName, err)
	}

	return &resp.BackupShortTermRetentionPolicy, nil
}

// GetSqlDbLTR retrieves the long term retention policy for an Azure SQL Database.
func (c *Client) GetSqlDbLTR(
	ctx context.Context,
	resourceGroup, serverName, databaseName string,
) (*armsql.LongTermRetentionPolicy, error) {
	client, err := armsql.NewLongTermRetentionPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create LTR client: %w", err)
	}

	resp, err := client.Get(ctx, resourceGroup, serverName, databaseName, armsql.LongTermRetentionPolicyNameDefault, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get LTR policy for db %q on server %q: %w", databaseName, serverName, err)
	}

	return &resp.LongTermRetentionPolicy, nil
}

// ApplySqlDbSTR upserts the short term retention policy for an Azure SQL Database.
func (c *Client) ApplySqlDbSTR(ctx context.Context, db azure.ResolvedSqlDb) error {
	client, err := armsql.NewBackupShortTermRetentionPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create STR client: %w", err)
	}

	params := armsql.BackupShortTermRetentionPolicy{
		Properties: &armsql.BackupShortTermRetentionPolicyProperties{
			RetentionDays:             to.Ptr(int32(db.Spec.ShortTermRetention.RetentionDays)),
			DiffBackupIntervalInHours: to.Ptr(armsql.DiffBackupIntervalInHours(db.Spec.ShortTermRetention.DifferentialBackupFrequency)),
		},
	}

	poller, err := client.BeginCreateOrUpdate(ctx, db.ResourceGroup, db.ServerName, db.ResourceName, armsql.ShortTermRetentionPolicyNameDefault, params, nil)
	if err != nil {
		return fmt.Errorf("failed to begin applying STR policy for db %q: %w", db.ResourceName, err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to apply STR policy for db %q: %w", db.ResourceName, err)
	}
	return nil
}

// ApplySqlDbLTR upserts the long term retention policy for an Azure SQL Database.
func (c *Client) ApplySqlDbLTR(ctx context.Context, db azure.ResolvedSqlDb) error {
	client, err := armsql.NewLongTermRetentionPoliciesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create LTR client: %w", err)
	}

	ltr := db.Spec.LongTermRetention

	// Azure requires WeekOfYear to be in [1,52] whenever the yearly-retention
	// field is present in the request — even when yearly retention is P0Y, as
	// happens when clearing an existing LTR policy (all retentions 0). WeekOfYear
	// is meaningless with no yearly backups, so send a valid placeholder purely
	// to satisfy the API. When yearly retention IS requested (>0), pass WeekOfYear
	// through unchanged (0 if the user omitted it) so Azure surfaces
	// LongTermRetentionMissingWeekOfYear — the intended validation error.
	weekOfYear := int32(ltr.WeekOfYear)
	if ltr.YearlyRetention == 0 && weekOfYear == 0 {
		weekOfYear = 1
	}

	params := armsql.LongTermRetentionPolicy{
		Properties: &armsql.BaseLongTermRetentionPolicyProperties{
			WeeklyRetention:  to.Ptr(fmt.Sprintf("P%dW", ltr.WeeklyRetention)),
			MonthlyRetention: to.Ptr(fmt.Sprintf("P%dM", ltr.MonthlyRetention)),
			YearlyRetention:  to.Ptr(fmt.Sprintf("P%dY", ltr.YearlyRetention)),
			WeekOfYear:       to.Ptr(weekOfYear),
		},
	}

	// NOTE: time-based immutability (ltr_immutability / lock_immutability) is
	// rejected at validate time (validateAzureSqlDbRules) while the SDK cannot
	// support it, so it never reaches this apply path.

	poller, err := client.BeginCreateOrUpdate(ctx, db.ResourceGroup, db.ServerName, db.ResourceName, armsql.LongTermRetentionPolicyNameDefault, params, nil)
	if err != nil {
		return fmt.Errorf("failed to begin applying LTR policy for db %q: %w", db.ResourceName, err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to apply LTR policy for db %q: %w", db.ResourceName, err)
	}
	return nil
}

func (c *Client) ApplySqlDbRedundancy(ctx context.Context, db azure.ResolvedSqlDb) error {
	client, err := armsql.NewDatabasesClient(c.SubscriptionID, c.Credential, nil)
	if err != nil {
		return fmt.Errorf("failed to create databases client: %w", err)
	}

	redundancyMap := map[string]armsql.BackupStorageRedundancy{
		"Local":   armsql.BackupStorageRedundancyLocal,
		"Zone":    armsql.BackupStorageRedundancyZone,
		"Geo":     armsql.BackupStorageRedundancyGeo,
		"GeoZone": armsql.BackupStorageRedundancyGeoZone,
	}

	redundancy, ok := redundancyMap[db.Spec.Redundancy]
	if !ok {
		return fmt.Errorf("unsupported redundancy %q", db.Spec.Redundancy)
	}

	poller, err := client.BeginUpdate(ctx, db.ResourceGroup, db.ServerName, db.ResourceName,
		armsql.DatabaseUpdate{
			Properties: &armsql.DatabaseUpdateProperties{
				RequestedBackupStorageRedundancy: to.Ptr(redundancy),
			},
		}, nil)
	if err != nil {
		return fmt.Errorf("failed to begin updating redundancy for db %q: %w", db.ResourceName, err)
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to update redundancy for db %q: %w", db.ResourceName, err)
	}
	return nil
}
