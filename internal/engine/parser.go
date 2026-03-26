package engine

// In internal/engine/parser.go implement ValidatePolicy(policy BackupPolicy, sourcePath string) error.
// Run go-playground/validator on the fully resolved struct.
// If validation fails wrap the error with the source file path so the message
//   reads: policies/production/finance.yaml: metadata.owner is required.
// Runs after resolution so it validates the merged result.

// Validates the a BackupPolicy against some requirements.
// Inputs: policy BackupPolicy, sourcePath string (the yaml file)
// func ValidatePolicy(policy BackupPolicy, sourcePath string) {

// }
