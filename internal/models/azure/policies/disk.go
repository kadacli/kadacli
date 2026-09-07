package policies

import "fmt"

// contains the "spec" of a disk policy in Azure
type AzureDiskSchedule struct {
	Frequency string `yaml:"frequency"  validate:"required,oneof=hourly daily"`
	Interval  int    `yaml:"interval"   validate:"omitempty,oneof=1 2 4 6 8 12"` // Only required if frequency == hourly
	Time      string `yaml:"time"       validate:"omitempty,datetime=15:04"`     // Only requred if frequency == daily
}

type AzureDiskRetention struct {
	Default        bool                     `yaml:"default"` // If default the rest is optional
	RetentionRules []AzureDiskRetentionRule `yaml:"retention_rules" validate:"omitempty,dive"`
}

type AzureDiskRetentionRule struct {
	RetentionRule     string `yaml:"rule" validate:"required,oneof=daily weekly"`
	RetentionDuration int    `yaml:"duration" validate:"omitempty,min=1,max=436"`
}

// TagName returns the Azure retention tag this rule maps to. Azure applies a
// retention rule only to backups carrying a tag of the same name, and its tag
// names are capitalised — a rule named "weekly" can never match the "Weekly"
// tag, so it would sit in the policy doing nothing.
func (r AzureDiskRetentionRule) TagName() string {
	if r.RetentionRule == "weekly" {
		return "Weekly"
	}
	return "Daily"
}

// ISODuration renders the rule's duration the way Azure stores it: weekly
// rules in weeks, daily rules in days.
func (r AzureDiskRetentionRule) ISODuration() string {
	if r.RetentionRule == "weekly" {
		return fmt.Sprintf("P%dW", r.RetentionDuration)
	}
	return fmt.Sprintf("P%dD", r.RetentionDuration)
}

// DefaultISODuration returns the duration of the mandatory "Default" retention
// rule. When retention.default is true it mirrors the declared rules (the last
// one wins); otherwise Azure's 7-day fallback stands. Both the policy builder
// and the differ read it from here so the two cannot drift apart.
func (r AzureDiskRetention) DefaultISODuration() string {
	duration := "P7D"
	if !r.Default {
		return duration
	}
	for _, rule := range r.RetentionRules {
		duration = rule.ISODuration()
	}
	return duration
}

type AzureDiskSpec struct {
	SnapshotResourceGroup string             `yaml:"snapshot_resource_group"` // optional, defaults to disk's resource group
	Schedule              AzureDiskSchedule  `yaml:"schedule" validate:"required"`
	Retention             AzureDiskRetention `yaml:"retention" validate:"required"`
}
