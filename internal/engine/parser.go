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
		errs = append(errs, errors.New("apiVersion is required "))
	} else if filledPolicy.ApiVersion != "backupctl/v1" {
		errs = append(errs, errors.New("apiVersion has an invalid value. The only allowed value is backupctl/v1 "))
	}

	// kind
	if filledPolicy.Kind == "" {
		errs = append(errs, errors.New("Kind is required "))
	}

	// metadata
	if filledPolicy.Metadata.Name == "" {
		errs = append(errs, errors.New("metadata.name is required "))
	}
	if filledPolicy.Metadata.Owner == "" {
		errs = append(errs, errors.New("metadata.owner is required "))
	}

	// spec
	if filledPolicy.Spec.Targets == nil {
		errs = append(errs, errors.New("spec.targets is required"))
	} else {
		for i, target := range filledPolicy.Spec.Targets {
			if target.Type == "" {
				errs = append(errs, fmt.Errorf("spec.targets[%d].type is required", i))
			} // TODO validate targets for DB
		}
	}
	// TODO: Validate timezone
	if filledPolicy.Spec.Timezone == "" {
		errs = append(errs, errors.New("Kind is required "))
	}

	// storage
	if filledPolicy.Spec.StorageProvider.Provider == "" {
		errs = append(errs, fmt.Errorf("provider is not set in storage"))
		return errs
	}
	if filledPolicy.Spec.StorageProvider.Vault == "" {
		errs = append(errs, fmt.Errorf("vault is not set in storage"))
	}
	if filledPolicy.Spec.StorageProvider.Resource_group == "" {
		errs = append(errs, fmt.Errorf("Resource_group is not set in storage"))
	}
	if filledPolicy.Spec.StorageProvider.BackupPolicy == "" {
		errs = append(errs, fmt.Errorf("backup_policy is not set in storage"))
	}
	if len(filledPolicy.Spec.StorageProvider.DiskSelection) == 0 {
		errs = append(errs, fmt.Errorf("disk_selection is not set is not set in storage"))
	}
	if len(filledPolicy.Spec.StorageProvider.DiskSelection) == 0 && !filledPolicy.Spec.StorageProvider.IncludeFutureDisks {
		errs = append(errs, fmt.Errorf("Not a valid combination of DiskSelection and IncludeFutureDisks"))
	}
	return errs
}
