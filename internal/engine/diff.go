package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// DiffPolicy compares a local BackupPolicy against the live Azure policy.
// Returns a slice of human-readable difference strings.
func DiffPolicy(local models.BackupPolicy, remote armrecoveryservicesbackup.ProtectedItemsClientGetResponse) (map[string]interface{}, map[string]interface{}, error) {

	b, err := json.MarshalIndent(remote.Properties, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	remotePolicyMap, err := parseJson(b)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing response from Azure: %w", err)
	}

	localPolicyMap := backupPolicyToMap(local)

	valid, diff, err := compareMaps(localPolicyMap, remotePolicyMap)

	return valid, diff, nil
}

// Things to compare:
//
// policyName
// policyId
// diskLunList
// isInclusionList
// Compares two maps and compares the values. Returns matching entries and differing entries.
func compareMaps(mapOne map[string]any, mapTwo map[string]any) (map[string]any, map[string]any, error) {
	if mapOne == nil || mapTwo == nil {
		return nil, nil, fmt.Errorf("neither map can be nil")
	}

	valid := make(map[string]any)
	diff := make(map[string]any)

	for k, v := range mapOne {
		if val, ok := mapTwo[k]; ok && reflect.DeepEqual(val, v) {
			valid[k] = v
		} else {
			diff[k] = v
		}
	}

	// Catch keys that exist in mapTwo but not in mapOne
	for k, v := range mapTwo {
		if _, ok := mapOne[k]; !ok {
			diff[k] = v
		}
	}

	return valid, diff, nil
}

// Takes a local backupPolicy and returns a map
func backupPolicyToMap(bp models.BackupPolicy) map[string]any {
	return map[string]any{
		"policyName":      bp.Spec.StorageProvider.BackupPolicy,
		"policyId":        nil,
		"diskLunList":     bp.Spec.StorageProvider.DiskSelection,
		"isInclusionList": bp.Spec.StorageProvider.IncludeFutureDisks,
	}
}

// Takes a json file and returns a map
func parseJson(data []byte) (map[string]interface{}, error) {
	var item models.AzureVMBackupPolicy
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"policyName":      item.PolicyName,
		"policyId":        item.PolicyId,
		"diskLunList":     item.ExtendedProperties.DiskExclusionProperties.DiskLunList,
		"isInclusionList": item.ExtendedProperties.DiskExclusionProperties.IsInclusionList,
	}, nil
}
