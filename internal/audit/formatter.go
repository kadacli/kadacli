package audit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// FormatText renders an AuditReport as human-readable terminal output.
func FormatText(r models.AuditReport) string {
	var sb strings.Builder

	line := strings.Repeat("─", 42)
	fmt.Fprintf(&sb, "Scanning subscription: %s\n\n", r.SubscriptionID)
	fmt.Fprintf(&sb, "Summary\n%s\n", line)
	fmt.Fprintf(&sb, "  Compliant        %3d resources\n", r.Summary.Compliant)
	fmt.Fprintf(&sb, "  Drifted          %3d resources\n", r.Summary.Drifted)
	fmt.Fprintf(&sb, "  Unprotected      %3d resources\n", r.Summary.Unprotected)
	fmt.Fprintf(&sb, "%s\n", line)

	if len(r.Drifted) > 0 {
		fmt.Fprintf(&sb, "\nDrifted Resources:\n")
		for _, dr := range r.Drifted {
			fmt.Fprintf(&sb, "  %-28s (%s, %s)\n",
				dr.Resource.Name, dr.Resource.Type, dr.Resource.ResourceGroup)
			for _, d := range dr.Diffs {
				fmt.Fprintf(&sb, "    %s: want %s, got %s\n", d.Field, d.Desired, d.Actual)
			}
		}
	}

	if len(r.Unprotected) > 0 {
		fmt.Fprintf(&sb, "\nUnprotected Resources:\n")
		for _, ur := range r.Unprotected {
			fmt.Fprintf(&sb, "  %-28s (%s, %s)\n", ur.Name, ur.Type, ur.ResourceGroup)
		}
	}

	fmt.Fprintf(&sb, "\nReport generated at: %s\n", r.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	return sb.String()
}

// FormatJSON renders an AuditReport as indented JSON.
func FormatJSON(r models.AuditReport) (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal audit report: %w", err)
	}
	return string(b), nil
}
