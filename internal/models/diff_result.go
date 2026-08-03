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

// HasChanges reports whether this target differs from its live state: a policy
// definition that does not exist yet, a definition whose fields have changed,
// or a protected item that needs to be (re)assigned all count.
//
// This is what `diff` exits 1 on, so it is defined once here rather than
// re-derived per provider — a provider that grows a new diff category must add
// it to this predicate or its drift will be reported as compliant.
func (d DiffResult) HasChanges() bool {
	return d.RetentionMissing || len(d.RetentionApply) > 0 || len(d.ToApply) > 0
}
