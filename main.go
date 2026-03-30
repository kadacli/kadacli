package main

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/cmd"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
)

func loadAndResolve() {
	errors, rawPolicies := engine.LoadAllPolicies("example_yaml")

	if len(errors) > 0 {
		// print errors
		for path, err := range errors {
			fmt.Println(fmt.Errorf("Error loading the policy: %s. \n the error is %w \n", path, err))
		}
	}

	backupPolicies := engine.Resolve(rawPolicies)

	// print backupPolicies
	for i, policy := range backupPolicies {
		fmt.Println(i, policy)
	}
}

func main() {
	cmd.Execute()
}
