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

// ─── Loader tests ─────────────────────────────────────────────────────────────

type loadTest struct {
	dir                 string
	numOfExpectedErrors int
}

var loadTests = []loadTest{
	{"../../yaml_files/valid_yaml", 0},
	{"../../yaml_files/invalid_yaml", 2},
}

func TestLoadAll(t *testing.T) {
	t.Log("Testing loader.go")

	for i, test := range loadTests {
		policies, assignments, errs := LoadAll(test.dir)
		numErrs := len(errs)

		if numErrs != test.numOfExpectedErrors {
			t.Errorf("test[%d]: expected %d error(s), got %d: %v",
				i, test.numOfExpectedErrors, numErrs, errs)
		} else {
			t.Logf("test[%d]: %d error(s) as expected; loaded %d policies, %d assignments",
				i, numErrs, len(policies), len(assignments))
		}
	}
}

// ─── Resolver tests ───────────────────────────────────────────────────────────

func makeBasePolicy(name string) models.Policy {
	return models.Policy{
		APIVersion: "backupctl/v1",
		Kind:       "Policy",
		Metadata: models.PolicyMetadata{
			Name:        name,
			Environment: "staging",
			Owner:       "devops@company.com",
		},
		Spec: models.PolicySpec{
			Timezone: "Europe/Stockholm",
			Vault: models.VaultRef{
				Name:          "my-vault",
				ResourceGroup: "rg-backups",
			},
			Schedule: models.ScheduleSpec{
				Full: &models.FullSchedule{Frequency: "daily", Time: "02:00"},
			},
			Retention: models.RetentionSpec{
				Daily: &models.DailyRetention{Count: 30},
			},
		},
	}
}

func makeBaseAssignment(policyName, subscriptionID string, targets []models.AssignmentTarget) models.Assignment {
	return models.Assignment{
		APIVersion: "backupctl/v1",
		Kind:       "Assignment",
		Metadata: models.AssignmentMetadata{
			Name:           "test-assignment",
			SubscriptionID: subscriptionID,
			Environment:    "staging",
			Owner:          "devops@company.com",
		},
		Spec: models.AssignmentSpec{
			Assignments: []models.PolicyAssignment{
				{Policy: policyName, Targets: targets},
			},
		},
	}
}

func TestResolve(t *testing.T) {
	t.Log("Testing resolver.go")

	validPolicy := makeBasePolicy("gold-backup-policy")
	validTarget := models.AssignmentTarget{
		Type:          "azure_vm",
		ResourceGroup: "rg-prod",
		ResourceName:  "vm-prod-web-01",
	}
	validAssignment := makeBaseAssignment("gold-backup-policy", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", []models.AssignmentTarget{validTarget})

	t.Run("valid policy and assignment resolves successfully", func(t *testing.T) {
		resolved, errs := Resolve([]models.Policy{validPolicy}, []models.Assignment{validAssignment})
		if len(errs) > 0 {
			t.Errorf("expected no errors, got: %v", errs)
		}
		if len(resolved) != 1 {
			t.Errorf("expected 1 resolved target, got %d", len(resolved))
		}
	})

	t.Run("missing policy reference returns error", func(t *testing.T) {
		badAssignment := makeBaseAssignment("nonexistent-policy", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", []models.AssignmentTarget{validTarget})
		_, errs := Resolve([]models.Policy{validPolicy}, []models.Assignment{badAssignment})
		if len(errs) == 0 {
			t.Error("expected error for unknown policy reference, got none")
		}
	})

	t.Run("invalid policy apiVersion returns error", func(t *testing.T) {
		bad := validPolicy
		bad.APIVersion = "wrong/v1"
		_, errs := Resolve([]models.Policy{bad}, []models.Assignment{validAssignment})
		if len(errs) == 0 {
			t.Error("expected error for invalid apiVersion, got none")
		}
	})

	t.Run("missing policy vault returns error", func(t *testing.T) {
		bad := validPolicy
		bad.Spec.Vault.Name = ""
		_, errs := Resolve([]models.Policy{bad}, []models.Assignment{validAssignment})
		if len(errs) == 0 {
			t.Error("expected error for missing vault name, got none")
		}
	})

	t.Run("overrides are applied to resolved target", func(t *testing.T) {
		override := models.AssignmentTarget{
			Type:          "azure_vm",
			ResourceGroup: "rg-prod",
			ResourceName:  "vm-prod-db-01",
			Overrides: &models.PolicyOverrides{
				Retention: &models.RetentionSpec{
					Daily: &models.DailyRetention{Count: 90},
				},
			},
		}
		a := makeBaseAssignment("gold-backup-policy", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", []models.AssignmentTarget{override})
		resolved, errs := Resolve([]models.Policy{validPolicy}, []models.Assignment{a})
		if len(errs) > 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if resolved[0].Policy.Spec.Retention.Daily.Count != 90 {
			t.Errorf("expected daily retention count 90, got %d", resolved[0].Policy.Spec.Retention.Daily.Count)
		}
	})
}

// ─── compareMaps tests ────────────────────────────────────────────────────────

type compareMapsTest struct {
	mapOne              map[string]any
	mapTwo              map[string]any
	equalSettings       map[string]any
	settingsToApply     map[string]any
	settingsToOverwrite map[string]any
	expectedError       error
}

var mapTests = []compareMapsTest{
	{
		mapOne:        makeNilMap(),
		mapTwo:        makeNilMap(),
		expectedError: fmt.Errorf("mapOne contains a nil value for key \"a\""),
	},
	{
		mapOne:              map[string]any{"policyName": "test1", "a": 1},
		mapTwo:              map[string]any{"policyName": "test1", "a": 2, "b": 3},
		equalSettings:       map[string]any{"policyName": "test1"},
		settingsToApply:     map[string]any{"a": 1},
		settingsToOverwrite: map[string]any{"a": 2},
		expectedError:       nil,
	},
	{
		mapOne:              map[string]any{"policyName": "test2"},
		mapTwo:              makeNilMap(),
		equalSettings:       map[string]any{},
		settingsToApply:     map[string]any{"policyName": "test2"},
		settingsToOverwrite: map[string]any{},
		expectedError:       nil,
	},
	{
		mapOne:        nil,
		mapTwo:        makeNilMap(),
		expectedError: fmt.Errorf("mapOne must not be nil or empty"),
	},
}

func TestCompareMaps(t *testing.T) {
	for _, test := range mapTests {
		t.Run("comparing maps", func(t *testing.T) {
			valid, diff, overwrite, err := compareMaps(test.mapOne, test.mapTwo)

			errorsMatch := (err == nil && test.expectedError == nil) ||
				(err != nil && test.expectedError != nil && err.Error() == test.expectedError.Error())
			if !errorsMatch {
				t.Errorf("expected error: %v, got: %v", test.expectedError, err)
			}

			if test.expectedError == nil {
				if !reflect.DeepEqual(valid, test.equalSettings) {
					t.Errorf("equal: expected %s, got %s", printMap(test.equalSettings), printMap(valid))
				}
				if !reflect.DeepEqual(diff, test.settingsToApply) {
					t.Errorf("diff: expected %s, got %s", printMap(test.settingsToApply), printMap(diff))
				}
				if !reflect.DeepEqual(overwrite, test.settingsToOverwrite) {
					t.Errorf("overwrite: expected %s, got %s", printMap(test.settingsToOverwrite), printMap(overwrite))
				}
			}
		})
	}
}

// ─── DiffPolicy tests ─────────────────────────────────────────────────────────

func makeRemoteResponse(policyName string, diskLunList []*int32, isInclusionList bool) armrecoveryservicesbackup.ProtectedItemsClientGetResponse {
	return armrecoveryservicesbackup.ProtectedItemsClientGetResponse{
		ProtectedItemResource: armrecoveryservicesbackup.ProtectedItemResource{
			Properties: &armrecoveryservicesbackup.AzureIaaSComputeVMProtectedItem{
				PolicyName: to.Ptr(policyName),
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

func makeLocalRT(policyName string, diskLUNs []int32, includeFuture bool) models.ResolvedTarget {
	return models.ResolvedTarget{
		Policy: models.Policy{
			Metadata: models.PolicyMetadata{Name: policyName},
			Spec: models.PolicySpec{
				Vault: models.VaultRef{Name: "my-vault", ResourceGroup: "rg-backups"},
			},
		},
		Target: models.AssignmentTarget{
			Type:               "azure_vm",
			ResourceGroup:      "rg-prod",
			ResourceName:       "vm-prod-01",
			DiskSelection:      diskLUNs,
			IncludeFutureDisks: includeFuture,
		},
	}
}

type diffPolicyTest struct {
	local                       models.ResolvedTarget
	remote                      armrecoveryservicesbackup.ProtectedItemsClientGetResponse
	expectedEqualSettings       map[string]any
	expectedSettingsToApply     map[string]any
	expectedSettingsToOverwrite map[string]any
	expectedError               error
}

var diffPolicyTests = []diffPolicyTest{
	{
		// All fields match — nothing to apply
		local:  makeLocalRT("daily-backup", []int32{0, 1}, true),
		remote: makeRemoteResponse("daily-backup", []*int32{to.Ptr(int32(0)), to.Ptr(int32(1))}, true),
		expectedEqualSettings: map[string]any{
			"policyName":      "daily-backup",
			"diskLunList":     []int32{0, 1},
			"isInclusionList": true,
		},
		expectedSettingsToApply:     map[string]any{},
		expectedSettingsToOverwrite: map[string]any{},
		expectedError:               nil,
	},
	{
		// Policy name differs — remote should be overwritten
		local:  makeLocalRT("daily-backup", []int32{0, 1}, true),
		remote: makeRemoteResponse("old-backup-policy", []*int32{to.Ptr(int32(0)), to.Ptr(int32(1))}, true),
		expectedEqualSettings: map[string]any{
			"diskLunList":     []int32{0, 1},
			"isInclusionList": true,
		},
		expectedSettingsToApply:     map[string]any{"policyName": "daily-backup"},
		expectedSettingsToOverwrite: map[string]any{"policyName": "old-backup-policy"},
		expectedError:               nil,
	},
	{
		// Disk list and inclusion flag differ
		local:  makeLocalRT("daily-backup", []int32{0, 1}, true),
		remote: makeRemoteResponse("daily-backup", []*int32{to.Ptr(int32(2)), to.Ptr(int32(3))}, false),
		expectedEqualSettings: map[string]any{
			"policyName": "daily-backup",
		},
		expectedSettingsToApply: map[string]any{
			"diskLunList":     []int32{0, 1},
			"isInclusionList": true,
		},
		expectedSettingsToOverwrite: map[string]any{
			"diskLunList":     []int32{2, 3},
			"isInclusionList": false,
		},
		expectedError: nil,
	},
	{
		// All fields differ
		local:                 makeLocalRT("daily-backup", []int32{0, 1}, true),
		remote:                makeRemoteResponse("old-policy", []*int32{to.Ptr(int32(2)), to.Ptr(int32(3))}, false),
		expectedEqualSettings: map[string]any{},
		expectedSettingsToApply: map[string]any{
			"policyName":      "daily-backup",
			"diskLunList":     []int32{0, 1},
			"isInclusionList": true,
		},
		expectedSettingsToOverwrite: map[string]any{
			"policyName":      "old-policy",
			"diskLunList":     []int32{2, 3},
			"isInclusionList": false,
		},
		expectedError: nil,
	},
}

func TestDiffPolicy(t *testing.T) {
	for i, test := range diffPolicyTests {
		t.Run(fmt.Sprintf("DiffPolicy test[%d]", i), func(t *testing.T) {
			equalSettings, settingsToApply, _, settingsToOverwrite, err := DiffPolicy(test.local, test.remote)

			errorsMatch := (err == nil && test.expectedError == nil) ||
				(err != nil && test.expectedError != nil && err.Error() == test.expectedError.Error())
			if !errorsMatch {
				t.Errorf("expected error: %v, got: %v", test.expectedError, err)
			}

			if test.expectedError == nil {
				if !reflect.DeepEqual(equalSettings, test.expectedEqualSettings) {
					t.Errorf("equalSettings: expected %s, got %s",
						printMap(test.expectedEqualSettings), printMap(equalSettings))
				}
				if !reflect.DeepEqual(settingsToApply, test.expectedSettingsToApply) {
					t.Errorf("settingsToApply: expected %s, got %s",
						printMap(test.expectedSettingsToApply), printMap(settingsToApply))
				}
				if !reflect.DeepEqual(settingsToOverwrite, test.expectedSettingsToOverwrite) {
					t.Errorf("settingsToOverwrite: expected %s, got %s",
						printMap(test.expectedSettingsToOverwrite), printMap(settingsToOverwrite))
				}
			}
		})
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func makeNilMap() map[string]any {
	return map[string]any{"a": nil, "b": nil, "c": nil}
}

func printMap(m map[string]any) string {
	if len(m) == 0 {
		return "{}"
	}
	var sb strings.Builder
	for k, v := range m {
		sb.WriteString(fmt.Sprintf("%s: %v\n", k, v))
	}
	return sb.String()
}
