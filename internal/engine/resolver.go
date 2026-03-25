package engine

// In internal/engine/resolver.go implement
// Resolve(policies []RawPolicy, defaults []PolicyDefaults) []BackupPolicy.
// Load all *.yaml files from _defaults/.
// Build a map of name → PolicyDefaults.
// For each policy with a non-empty Metadata.Inherits field find
// the matching defaults and deep merge the PartialSpec (defaults fill gaps, policy-level values always win).
// This must be a pure function — no file I/O.
