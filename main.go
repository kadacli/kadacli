package main

import (
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
)

func main() {
	errors, rawPolicies := engine.LoadAllPolicies("example_yaml")

	if len(errors) > 0 {
		// print errors
		for _, e := range errors {
			fmt.Errorf("Error loading some policy: %w", e)
		}
	}

	backupPolicies := engine.Resolve(rawPolicies)

	// print backupPolicies
	for i, policy := range backupPolicies {
		fmt.Println(i, policy)
	}

}
