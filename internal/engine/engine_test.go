package engine

import (
	"testing"
)

// Unit tests for the engine
type loadTest struct {
	dir                 string
	numOfExpectedErrors int
}

var loadTests = []loadTest{
	{"../../yaml_files/good_yaml", 0},
	//{"../../yaml_files/bad_yaml", 2},
}

func TestLoadAllPolicies(t *testing.T) {

	for i, test := range loadTests {
		receivedErrors, rawPolicies := LoadAllPolicies(test.dir)
		numOfReceivedErrors := len(receivedErrors)

		if numOfReceivedErrors != test.numOfExpectedErrors {
			t.Errorf("numOfReceivedErrors %d differs from numOfExpectedErrors %d in TestLoadAllPolicies for test[%d]", numOfReceivedErrors, test.numOfExpectedErrors, i)
		} else {
			t.Logf("Successfully loaded these policies in TestLoadAllPolicies in test[%d]: \n", i)
			for i, _ := range rawPolicies {
				t.Log(rawPolicies[i].Metadata.Name)
			}
		}

	}

}
