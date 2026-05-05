package audit

import (
	"context"
	"fmt"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// Scanner discovers all backup-eligible resources in a subscription.
// TODO: implement for 1.0.0
type Scanner struct {
	SubscriptionID string
}

// ListResources returns all Azure resources of the given type in the subscription.
func (s *Scanner) ListResources(ctx context.Context, resourceType string) ([]models.Resource, error) {
	return nil, fmt.Errorf("scanner.ListResources not yet implemented")
}
