package engine

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// Unit tests for the engine.loader
type loadTest struct {
	dir                 string
	numOfExpectedErrors int
}

var loadTests = []loadTest{
	{"../../yaml_files/valid_yaml", 0},
	{"../../yaml_files/invalid_yaml", 2},
}

func TestLoadAllPolicies(t *testing.T) {
	t.Log("Testing loader.go")

	for i, test := range loadTests {
		rawPolicies, receivedErrors := LoadAllPolicies(test.dir)
		numOfReceivedErrors := len(receivedErrors)

		if numOfReceivedErrors != test.numOfExpectedErrors {
			t.Errorf("numOfReceivedErrors %d differs from numOfExpectedErrors %d in TestLoadAllPolicies for test[%d]",
				numOfReceivedErrors, test.numOfExpectedErrors, i)

		} else {
			t.Logf("numOfExpectedErrors: %d equals numOfReceivedErrors %d",
				test.numOfExpectedErrors, numOfReceivedErrors)
			for i := range rawPolicies {
				t.Log(rawPolicies[i].Metadata.Name)
			}
		}

	}
}

type resolverTest struct {
	policy              models.RawPolicy
	numOfExpectedErrors int
}

func TestResolver(t *testing.T) {
	t.Log("Testing resolver.go")

	validPolicy := models.RawPolicy{
		ApiVersion: "backupctl/v1",
		Kind:       "backupPolicy",
		Metadata: models.Metadata{
			Name:        "test-policy",
			Environment: "staging",
			Owner:       "team-platform",
		},
		Spec: models.Spec{
			Timezone: "Europe/Stockholm",
			Targets: []models.Target{
				{
					Type: "azure-vm",
					Selectors: []models.AzureSelector{
						{
							ResourceGroup: "rg-test",
							ResourceName:  "vm-test",
						},
					},
				},
			},
			StorageProvider: models.StorageProvider{
				Provider:           "azure",
				Vault:              "my-vault",
				Resource_group:     "rg-bpac",
				BackupPolicy:       "daily-backup",
				DiskSelection:      []int32{0, 1},
				IncludeFutureDisks: true,
			},
		},
	}

	// Creating an invalid RawPolicy
	invalidPolicies := createInvalidPolicies(validPolicy)
	for _, invalidPolicy := range invalidPolicies {
		resolverTester(t, resolverTest{
			policy:              invalidPolicy,
			numOfExpectedErrors: 1,
		})
	}

	resolverTester(t, resolverTest{
		policy:              validPolicy,
		numOfExpectedErrors: 0,
	})

}

// helper function.
// Takes the testing module and a resolverTest
func resolverTester(t *testing.T, test resolverTest) {
	// The Resolve function takes a slice, so we create a slice with len == 1
	var policy []models.RawPolicy
	policy = append(policy, test.policy)

	t.Logf("Testing with a valid policy: %v", test.numOfExpectedErrors == 0)

	backupPolices, errs := Resolve(policy)
	if len(errs) != test.numOfExpectedErrors {
		t.Errorf("The number of expected errors %d differs from the number of received Errors %d",
			len(errs), test.numOfExpectedErrors)
	} else if test.numOfExpectedErrors > 0 && len(backupPolices) > 0 {
		t.Errorf("Received a backupPolicy, but expected none")
	} else {
		t.Logf("Test passed")
	}

}

// Function to create all of the test cases for the invalid policies
func createInvalidPolicies(validPolicy models.RawPolicy) []models.RawPolicy {
	var invalidPolicies []models.RawPolicy

	// Missing apiVersion
	p1 := validPolicy
	p1.ApiVersion = ""
	invalidPolicies = append(invalidPolicies, p1)

	// Wrong apiVersion
	p2 := validPolicy
	p2.ApiVersion = "wrong/v1"
	invalidPolicies = append(invalidPolicies, p2)

	// Missing metadata.name
	p3 := validPolicy
	p3.Metadata.Name = ""
	invalidPolicies = append(invalidPolicies, p3)

	// Missing metadata.owner
	p4 := validPolicy
	p4.Metadata.Owner = ""
	invalidPolicies = append(invalidPolicies, p4)

	// Missing spec.targets
	p5 := validPolicy
	p5.Spec.Targets = nil
	invalidPolicies = append(invalidPolicies, p5)

	// Azure target with no selectors
	p6 := validPolicy
	p6.Spec.Targets = []models.Target{
		{Type: "azure-vm", Selectors: nil},
	}
	invalidPolicies = append(invalidPolicies, p6)

	// Azure target with selector missing resource_group
	p7 := validPolicy
	p7.Spec.Targets = []models.Target{
		{Type: "azure-vm", Selectors: []models.AzureSelector{
			{ResourceGroup: "", ResourceName: "vm-test"},
		}},
	}
	invalidPolicies = append(invalidPolicies, p7)

	// Missing spec.timezone
	p8 := validPolicy
	p8.Spec.Timezone = ""
	invalidPolicies = append(invalidPolicies, p8)

	// Empty spec.storage
	p9 := validPolicy
	p9.Spec.StorageProvider = models.StorageProvider{}
	invalidPolicies = append(invalidPolicies, p9)

	// Invalid policy_subtype
	p10 := validPolicy
	p10.Spec.StorageProvider = models.StorageProvider{
		Provider:      "azure",
		Vault:         "my-vault",
		BackupPolicy:  "daily-backup",
		DiskSelection: []int32{0, 1},
	}
	invalidPolicies = append(invalidPolicies, p10)

	return invalidPolicies
}

// compareMapsTest struct holds test cases and expected results
type compareMapsTest struct {
	mapOne          map[string]any
	mapTwo          map[string]any
	equalSettings   map[string]any
	settingsToApply map[string]any
	expectedError   error
}

// ISSUE: Tests are wrong
var mapTests = []compareMapsTest{
	{
		mapOne:          makeEmptyMap(),
		mapTwo:          makeEmptyMap(),
		equalSettings:   nil,
		settingsToApply: nil,
		expectedError:   fmt.Errorf("mapOne contains a nil value for key \"a\""),
	},
	{
		mapOne:          map[string]any{"policyName": "test1", "a": 1},
		mapTwo:          map[string]any{"policyName": "test1", "a": 2, "b": 3},
		equalSettings:   map[string]any{"policyName": "test1"},
		settingsToApply: map[string]any{"a": 1},
		expectedError:   nil,
	},
	{
		mapOne:          map[string]any{"policyName": "test2"},
		mapTwo:          makeEmptyMap(), // {"a": nil, "b": nil, "c": nil}
		equalSettings:   map[string]any{},
		settingsToApply: map[string]any{"policyName": "test2"},
		expectedError:   nil,
	},
	{
		mapOne:          nil,
		mapTwo:          makeEmptyMap(),
		equalSettings:   nil,
		settingsToApply: nil,
		expectedError:   fmt.Errorf("mapOne must not be nil"),
	},
}

// Test compares two maps and verifies the comparison functions work correctly
func TestCompareMaps(t *testing.T) {
	for _, test := range mapTests {
		t.Run(`Comparing maps`, func(t *testing.T) {
			valid, diff, err := compareMaps(test.mapOne, test.mapTwo)

			// Check error matches expectation
			errorsMatch := (err == nil && test.expectedError == nil) ||
				(err != nil && test.expectedError != nil && err.Error() == test.expectedError.Error())
			if !errorsMatch {
				t.Errorf("expected error: %v, but got: %v", test.expectedError, err)
			}

			// Only check valid/diff when no error is expected
			if test.expectedError == nil {
				if !reflect.DeepEqual(valid, test.equalSettings) {
					t.Errorf("expected matching map:\n%s\nbut got:\n%s", printMap(test.equalSettings), printMap(valid))
				}
				if !reflect.DeepEqual(diff, test.settingsToApply) {
					t.Errorf("expected difference map:\n%s\nbut got:\n%s", printMap(test.settingsToApply), printMap(diff))
				}
			}
		})
	}
}

// makeEmptyMap returns a map with nil values for all keys
func makeEmptyMap() map[string]any {
	return map[string]any{
		"a": nil,
		"b": nil,
		"c": nil,
	}
}

// printMap prints the given map in a readable format
func printMap(mapToPrint map[string]any) string {
	if len(mapToPrint) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Map contents:\n")
	for k, v := range mapToPrint {
		sb.WriteString(fmt.Sprintf("%s: %v\n", k, v))
	}
	return sb.String()
}

func makeRemoteResponse(policyName string, policyId string, diskLunList []*int32, isInclusionList bool) armrecoveryservicesbackup.ProtectedItemsClientGetResponse {
	return armrecoveryservicesbackup.ProtectedItemsClientGetResponse{
		ProtectedItemResource: armrecoveryservicesbackup.ProtectedItemResource{
			Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
				PolicyName: to.Ptr(policyName),
				PolicyID:   to.Ptr(policyId),
				ExtendedProperties: &armrecoveryservicesbackup.ExtendedProperties{
					DiskExclusionProperties: &armrecoveryservicesbackup.DiskExclusionProperties{
						DiskLunList:     diskLunList,
						IsInclusionList: to.Ptr(isInclusionList),
					},
				},
			},
		},
	}
}

type diffPolicyTest struct {
	local                   models.BackupPolicy
	remote                  armrecoveryservicesbackup.ProtectedItemsClientGetResponse
	expectedEqualSettings   map[string]any
	expectedSettingsToApply map[string]any
	expectedError           error
}

var diffPolicyTests = []diffPolicyTest{
	{
		// All fields match — nothing to apply
		local: models.BackupPolicy{
			Spec: models.Spec{
				StorageProvider: models.StorageProvider{
					BackupPolicy:       "daily-backup",
					DiskSelection:      []int32{0, 1},
					IncludeFutureDisks: true,
				},
			},
		},
		remote: makeRemoteResponse("daily-backup", "policy-123", []*int32{to.Ptr(int32(0)), to.Ptr(int32(1))}, true),
		expectedEqualSettings: map[string]any{
			"policyName":      "daily-backup",
			"diskLunList":     []int32{0, 1},
			"isInclusionList": true,
		},
		expectedSettingsToApply: map[string]any{},
		expectedError:           nil,
	},
	{
		// Policy name differs — remote should be overwritten
		local: models.BackupPolicy{
			Spec: models.Spec{
				StorageProvider: models.StorageProvider{
					BackupPolicy:       "daily-backup",
					DiskSelection:      []int32{0, 1},
					IncludeFutureDisks: true,
				},
			},
		},
		remote: makeRemoteResponse("old-backup-policy", "policy-123", []*int32{to.Ptr(int32(0)), to.Ptr(int32(1))}, true),
		expectedEqualSettings: map[string]any{
			"diskLunList":     []int32{0, 1},
			"isInclusionList": true,
		},
		expectedSettingsToApply: map[string]any{
			"policyName": "daily-backup",
		},
		expectedError: nil,
	},
	{
		// Disk list and inclusion flag differ
		local: models.BackupPolicy{
			Spec: models.Spec{
				StorageProvider: models.StorageProvider{
					BackupPolicy:       "daily-backup",
					DiskSelection:      []int32{0, 1},
					IncludeFutureDisks: true,
				},
			},
		},
		remote: makeRemoteResponse("daily-backup", "policy-123", []*int32{to.Ptr(int32(2)), to.Ptr(int32(3))}, false),
		expectedEqualSettings: map[string]any{
			"policyName": "daily-backup",
		},
		expectedSettingsToApply: map[string]any{
			"diskLunList":     []int32{0, 1},
			"isInclusionList": true,
		},
		expectedError: nil,
	},
	{
		// All fields differ => apply everything in local
		local: models.BackupPolicy{
			Spec: models.Spec{
				StorageProvider: models.StorageProvider{
					BackupPolicy:       "daily-backup",
					DiskSelection:      []int32{0, 1},
					IncludeFutureDisks: true,
				},
			},
		},
		remote:                makeRemoteResponse("old-backup-policy", "policy-123", []*int32{to.Ptr(int32(2)), to.Ptr(int32(3))}, false),
		expectedEqualSettings: map[string]any{},
		expectedSettingsToApply: map[string]any{
			"policyName":      "daily-backup",
			"diskLunList":     []int32{0, 1},
			"isInclusionList": true,
		},
		expectedError: nil,
	},
}

func TestDiffPolicy(t *testing.T) {
	for i, test := range diffPolicyTests {
		t.Run(fmt.Sprintf("DiffPolicy test[%d]", i), func(t *testing.T) {
			equalSettings, settingsToApply, _, err := DiffPolicy(test.local, test.remote)

			errorsMatch := (err == nil && test.expectedError == nil) ||
				(err != nil && test.expectedError != nil && err.Error() == test.expectedError.Error())
			if !errorsMatch {
				t.Errorf("expected error: %v, but got: %v", test.expectedError, err)
			}

			if test.expectedError == nil {
				if !reflect.DeepEqual(equalSettings, test.expectedEqualSettings) {
					t.Errorf("expected expectedEqualSettings map:\n%s\nbut got:\n%s", printMap(test.expectedEqualSettings), printMap(equalSettings))
				}
				if !reflect.DeepEqual(settingsToApply, test.expectedSettingsToApply) {
					t.Errorf("expected expectedSettingsToApply map:\n%s\nbut got:\n%s", printMap(test.expectedSettingsToApply), printMap(settingsToApply))
				}
			}
		})
	}
}
