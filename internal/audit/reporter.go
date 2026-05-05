package audit

import (
	"time"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// BuildReport assembles an AuditReport from compliant, drifted, and unprotected results.
// TODO: implement full drift detection for 1.0.0
func BuildReport(
	subscriptionID string,
	compliant []models.AuditResult,
	drifted []models.AuditResult,
	unprotected []models.Resource,
) models.AuditReport {
	return models.AuditReport{
		SubscriptionID: subscriptionID,
		GeneratedAt:    time.Now().UTC(),
		Summary: models.AuditSummary{
			Total:       len(compliant) + len(drifted) + len(unprotected),
			Compliant:   len(compliant),
			Drifted:     len(drifted),
			Unprotected: len(unprotected),
		},
		Compliant:   compliant,
		Drifted:     drifted,
		Unprotected: unprotected,
	}
}
