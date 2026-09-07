package models

import "testing"

// A target counted as compliant when it is not is the worst failure this
// predicate can have: `diff` would exit 0 and a pipeline would report a
// drifted environment as in sync.
func TestDiffResultHasChanges(t *testing.T) {
	tests := []struct {
		name string
		in   DiffResult
		want bool
	}{
		{
			name: "empty result is compliant",
			in:   DiffResult{},
			want: false,
		},
		{
			name: "equal-only fields are compliant",
			in: DiffResult{
				Equal:          map[string]any{"policyName": "gold"},
				RetentionEqual: map[string]any{"retention.daily.count": int32(30)},
			},
			want: false,
		},
		{
			name: "missing policy definition is a change",
			in:   DiffResult{RetentionMissing: true},
			want: true,
		},
		{
			name: "changed policy definition is a change",
			in:   DiffResult{RetentionApply: map[string]any{"retention.daily.count": int32(30)}},
			want: true,
		},
		{
			name: "changed protected item is a change",
			in:   DiffResult{ToApply: map[string]any{"policyName": "gold"}},
			want: true,
		},
		{
			name: "empty (non-nil) change maps are compliant",
			in: DiffResult{
				ToApply:        map[string]any{},
				RetentionApply: map[string]any{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.HasChanges(); got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}
