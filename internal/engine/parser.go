package engine

import (
	"errors"
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// In internal/engine/parser.go implement ValidatePolicy(policy BackupPolicy, sourcePath string) error.
// Run go-playground/validator on the fully resolved struct.
// If validation fails wrap the error with the source file path so the message
//   reads: policies/production/finance.yaml: metadata.owner is required.
// Runs after resolution so it validates the merged result.

// Checks the required fields.
// A policy yaml file can have many errors, therefore we return a slice of errors

// TODOs:
// Make validation extensible
// Give access to the file-path that is failing

func ValidatePolicy(filledPolicy models.RawPolicy) []error {
	var errs []error

	// What does it mean that a field is valid?
	// 1. The field is set if required
	// 2. The field has a valid value

	// apiVersion
	if filledPolicy.ApiVersion == "" {
		errs = append(errs, errors.New("apiVersion is required \n"))
	}
	if filledPolicy.ApiVersion != "backupctl/v1" {
		errs = append(errs, errors.New("apiVersion has an invalid value.\n The only allowed value is backupctl/v1 \n"))
	}

	// metadata
	if filledPolicy.Metadata.Name == "" {
		errs = append(errs, errors.New("metadata.name is required \n"))
	}
	if filledPolicy.Metadata.Owner == "" {
		errs = append(errs, errors.New("metadata.owner is required \n"))
	}

	// spec
	if filledPolicy.Spec.Targets == nil {
		errs = append(errs, errors.New("spec.targets is required \n"))
	} else {
		for i, target := range filledPolicy.Spec.Targets {
			if target.Type == "" {
				errs = append(errs, fmt.Errorf("type is required in spec.targets[%d] \n", i-1))
			}
			if target.Selectors == nil {
				errs = append(errs, fmt.Errorf("selectors is required in spec.targets[%d] \n", i-1))
			}
		}
	}

	// TODO: Validate schedule

	// TODO: Validate timezone

	// storage
	if len(filledPolicy.Spec.StorageProviders) == 0 {
		errs = append(errs, errors.New("spec.storage has a length of 0. Must be set. \n"))
	} else {
		for i, s := range filledPolicy.Spec.StorageProviders {
			if s.Provider == "" {
				errs = append(errs, fmt.Errorf("provider is not set in storage[%d] \n", i-1))
			}
			if s.Vault == "" {
				errs = append(errs, fmt.Errorf("vault is not set in storage[%d] \n", i-1))
			}
			if s.Redundancy == "" {
				errs = append(errs, fmt.Errorf("redundancy is not set in storage[%d] \n", i-1))
			}
		}
	}

	fmt.Printf("Number of errors: %d \n", len(errs))
	return errs
}
