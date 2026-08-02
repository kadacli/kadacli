package models

// DiffResult holds the outcome of a single target diff for any provider
type DiffResult struct {
	Target             any
	Equal              map[string]any
	ToApply            map[string]any
	Overwrite          map[string]any
	RetentionEqual     map[string]any
	RetentionApply     map[string]any
	RetentionOverwrite map[string]any
	RetentionMissing   bool
}
