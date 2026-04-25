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
// Requirement: Has to return where mapOne (localPolicy) differs from mapTwo (remotePolicy) - because remotePolicy shall be overwritten
func compareMaps(mapOne map[string]any, mapTwo map[string]any) (map[string]any, map[string]any, error) {
	if mapOne == nil || mapTwo == nil {
		return nil, nil, fmt.Errorf("ERROR: neither map can be nil")
	} else if allValuesNil(mapOne) {
		return nil, nil, fmt.Errorf("ERROR: mapOne cannot be empty")
	}

	valid := make(map[string]any)
	diff := make(map[string]any)

	// Check all mapTwo entries against mapOne
	for k, v := range mapTwo {
		if val, ok := mapOne[k]; ok && reflect.DeepEqual(val, v) {
			valid[k] = v
		} else {
			diff[k] = v // mapTwo's value — this is what gets overwritten
		}
	}

	return valid, diff, nil
}

// Takes a local backupPolicy and returns a map
func backupPolicyToMap(bp models.BackupPolicy) map[string]any {
	return map[string]any{
		"policyName":      bp.Spec.StorageProvider.BackupPolicy,
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

	// Dereference []*int32 to []int32 so it matches the local policy type
	diskLuns := make([]int32, len(item.ExtendedProperties.DiskExclusionProperties.DiskLunList))
	for i, ptr := range item.ExtendedProperties.DiskExclusionProperties.DiskLunList {
		if ptr != nil {
			diskLuns[i] = *ptr
		}
	}

	return map[string]interface{}{
		"policyName":      item.PolicyName,
		"diskLunList":     diskLuns,
		"isInclusionList": item.ExtendedProperties.DiskExclusionProperties.IsInclusionList,
	}, nil
}

// helper function to check if all the values of a map are empty
func allValuesNil(m map[string]interface{}) bool {
	for _, v := range m {
		if v != nil {
			return false
		}
	}
	return true
}
