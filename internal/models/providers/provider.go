package providers

import (
	"context"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// internal/providers/provider.go
type Provider interface {
	// ListResources returns all resources of a given type in the subscription
	ListResources(ctx context.Context, resourceType string) ([]Resource, error)

	// GetPolicy returns the currently applied backup policy for a resource
	GetPolicy(ctx context.Context, resource Resource) (*models.Policy, error)

	// ApplyPolicy applies a backup policy to a resource
	ApplyPolicy(ctx context.Context, resource Resource, policy *models.Policy) error

	// RemovePolicy removes backup protection from a resource
	RemovePolicy(ctx context.Context, resource Resource) error
}
