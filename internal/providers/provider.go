package providers

import (
	"context"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// Provider is the interface all cloud backup providers must implement.
// Azure implements this for 1.0.0; AWS is scaffolded for a future release.
type Provider interface {
	ListResources(ctx context.Context, resourceType string) ([]models.Resource, error)
	GetPolicy(ctx context.Context, resource models.Resource) (*models.Policy, error)
	ApplyPolicy(ctx context.Context, resource models.Resource, policy *models.Policy) error
	RemovePolicy(ctx context.Context, resource models.Resource) error
}
