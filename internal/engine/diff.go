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
// Returns a map with the settings to be applied
// a map with the settings that are already identical
// and a map that represents what the final settings will look like. (useful for --force or --dry-run flags.)
func DiffPolicy(local models.BackupPolicy, remote armrecoveryservicesbackup.ProtectedItemsClientGetResponse) (map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	b, err := json.MarshalIndent(remote.Properties, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	remotePolicyMap, err := parseJson(b)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error parsing response from Azure: %w", err)
	}

	localPolicyMap := backupPolicyToMap(local)

	equalSettings, settingsToApply, err := compareMaps(localPolicyMap, remotePolicyMap)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ERROR: Error when comparing maps: %v", err)
	}

	return equalSettings, settingsToApply, localPolicyMap, nil
}

// Things to compare:
//
// policyName
// diskLunList
// isInclusionList
// Compares two maps and compares the values. Returns matching entries and differing entries.
// Requirement: Has to return where mapOne (localPolicy) differs from mapTwo (remotePolicy) - because remotePolicy shall be overwritten
func compareMaps(mapOne, mapTwo map[string]any) (equal map[string]any, diff map[string]any, err error) {
	if mapOne == nil {
		return nil, nil, fmt.Errorf("mapOne must not be nil")
	}
	for k, v := range mapOne {
		if v == nil {
			return nil, nil, fmt.Errorf("mapOne contains a nil value for key %q", k)
		}
	}

	equal = make(map[string]any)
	diff = make(map[string]any)

	for key, valueOne := range mapOne {
		valueTwo, exists := mapTwo[key]

		if !exists {
			// Key is missing from mapTwo — needs to be added
			diff[key] = valueOne
			continue
		}

		if reflect.DeepEqual(valueOne, valueTwo) {
			// Values are identical — part of the overlap
			equal[key] = valueOne
		} else {
			// Key exists but value differs — needs to be updated
			diff[key] = valueOne
		}
	}

	return equal, diff, nil
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
