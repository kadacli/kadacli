package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// Client holds authenticated Azure credentials used by all Azure provider operations.
type Client struct {
	Credential     *azidentity.DefaultAzureCredential
	SubscriptionID string
}

// NewClient creates an authenticated Azure client using DefaultAzureCredential.
func NewClient(subscriptionID string) (*Client, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain Azure credentials: %w", err)
	}
	return &Client{
		Credential:     cred,
		SubscriptionID: subscriptionID,
	}, nil
}
