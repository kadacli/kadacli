package provider

// external/provider.go

import "context"

// Provider is the interface all cloud backup providers must implement.
// It operates on already-resolved, provider-specific data — the engine
// layer is responsible for parsing, validating, and resolving policies
// before handing them to a Provider.
// NOTE: MUST remain vendor agnostic
type Provider interface {
	// ListResources returns all resources of a given kind in the subscription.
	// e.g. resourceKind = "vm", "disk", "sqlvm"
	ListResources(ctx context.Context, resourceKind string) ([]string, error)

	// GetPolicy fetches the current live policy for the target.
	GetPolicy(ctx context.Context) error

	// DiffPolicy compares the local resolved policy against the live state.
	// Used for dry-run and idempotency checks.
	DiffPolicy(ctx context.Context) (equal, toApply, local, toOverwrite map[string]interface{}, err error)

	// ApplyPolicy creates or updates the backup policy for the target.
	ApplyPolicy(ctx context.Context) error

	// RemovePolicy removes the backup policy from the target.
	RemovePolicy(ctx context.Context) error
}
