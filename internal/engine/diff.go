package engine

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// DiffPolicy compares a local BackupPolicy against the live Azure policy.
// Returns a slice of human-readable difference strings.
func DiffPolicy(local *models.BackupPolicy, remote armrecoveryservicesbackup.ProtectionPoliciesClientGetResponse) ([]string, error) {
	var diffs []string

	b, err := json.MarshalIndent(remote.Properties, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	return diffs, nil
}
