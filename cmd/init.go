package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <kind> <name>",
	Short: "Scaffold a new Policy or Assignment YAML file",
	Long: `Generates a starter YAML file for the given kind (Policy or Assignment).

Examples:
  bpac init Policy gold-vm-daily
  bpac init Assignment prod-vm-assignments`,
	Args: cobra.ExactArgs(2),
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	kind := args[0]
	name := args[1]

	var content string
	switch strings.ToLower(kind) {
	case "policy":
		content = policyTemplate(name)
	case "assignment":
		content = assignmentTemplate(name)
	default:
		return fmt.Errorf("unknown kind %q — must be Policy or Assignment", kind)
	}

	filename := name + ".yaml"
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file %q already exists", filename)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %q: %w", filename, err)
	}

	abs, _ := filepath.Abs(filename)
	fmt.Printf("Created %s\n", abs)
	return nil
}

func policyTemplate(name string) string {
	return fmt.Sprintf(`apiVersion: backupctl/v1
kind: Policy
provider: azure
metadata:
  name: %s
  environment: production
  timezone: Europe/Stockholm
  owner: ops@company.com

# WHERE the backup lives — vault and subscription
resource:
  subscription_id: "00000000-0000-0000-0000-000000000000"
  type: azure_vm
  vault:
    name: rsv-prod-backup
    resource_group: rg-backup-vault

# HOW the backup is configured — retention and schedule
spec:
  instant_rp_retention_days: 2
  retention:
    daily:
      count: 30
    weekly:
      day: Sunday
      count: 12
    monthly:
      week: Last
      day: Sunday
      count: 12
    yearly:
      month: January
      week: First
      day: Sunday
      count: 3
  schedule:
    full:
      frequency: daily
      time: "02:00"
`, name)
}

func assignmentTemplate(name string) string {
	return fmt.Sprintf(`apiVersion: backupctl/v1
kind: Assignment
provider: azure
metadata:
  name: %s
  environment: production
  owner: ops@company.com

# Vault and subscription are inherited from the referenced policy
assignments:
  - policy: <policy-name>
    targets:
      - resource_group: rg-workload-prod
        resource_name: vm-example-01
      # Optional: restrict which disks are backed up
      # - resource_group: rg-workload-prod
      #   resource_name: vm-example-02
      #   disk_selection:
      #     mode: include   # include | exclude | all
      #     luns: [0, 1]
`, name)
}
