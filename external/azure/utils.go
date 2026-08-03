package azure_external

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// utils.go holds the tools that is needed accross all azure functions and resources

func Authenticate() (*azidentity.DefaultAzureCredential, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to obtain credentials: %w", err)
	}
	return cred, nil
}

// Client holds authenticated Azure credentials used by all Azure provider operations.
type Client struct {
	Credential     *azidentity.DefaultAzureCredential
	SubscriptionID string
}
