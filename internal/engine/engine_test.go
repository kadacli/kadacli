package engine

import (
	"testing"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// Unit tests for the engine.loader
type loadTest struct {
	dir                 string
	numOfExpectedErrors int
}

var loadTests = []loadTest{
	{"../../yaml_files/good_yaml", 0},
	{"../../yaml_files/invalid_yaml", 2},
}

func TestLoadAllPolicies(t *testing.T) {
	t.Log("Testing loader.go")

	for i, test := range loadTests {
		rawPolicies, receivedErrors := LoadAllPolicies(test.dir)
		numOfReceivedErrors := len(receivedErrors)

		if numOfReceivedErrors != test.numOfExpectedErrors {
			t.Errorf("numOfReceivedErrors %d differs from numOfExpectedErrors %d in TestLoadAllPolicies for test[%d]", numOfReceivedErrors, test.numOfExpectedErrors, i)
		} else {
			t.Logf("numOfExpectedErrors: %d equals numOfReceivedErrors %d", test.numOfExpectedErrors, numOfReceivedErrors)
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
		Kind:       "BackupPolicy",
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
			StorageProviders: []models.StorageProvider{
				{
					Provider:           "azure",
					Vault:              "my-vault",
					PolicySubtype:      "Enhanced",
					BackupPolicy:       "daily-backup",
					DiskSelection:      []uint8{0, 1},
					IncludeFutureDisks: true,
				},
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
	p9.Spec.StorageProviders = []models.StorageProvider{}
	invalidPolicies = append(invalidPolicies, p9)

	// Invalid policy_subtype
	p10 := validPolicy
	p10.Spec.StorageProviders = []models.StorageProvider{
		{
			Provider:      "azure",
			Vault:         "my-vault",
			PolicySubtype: "Invalid",
			BackupPolicy:  "daily-backup",
			DiskSelection: []uint8{0, 1},
		},
	}
	invalidPolicies = append(invalidPolicies, p10)

	return invalidPolicies
}
