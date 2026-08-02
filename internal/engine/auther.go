package engine

import (
	"fmt"

	azure_external "github.com/dbjorklund/backup-policy-as-code/external/azure"
)

// authenticate to the given provider
// Returns the given provider's cred data type
func Authenticate(provider string) (any, error) {

	switch provider {
	case "azure":
		return azure_external.Authenticate()
	case "aws":
		// TODO: aws_external.Authenticate()
		return nil, fmt.Errorf("aws is not yet supported")
	default:
		return nil, fmt.Errorf("Provider %s is not yet supported", provider)
	}

}
