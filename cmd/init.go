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
  bpac init Policy gold-backup-policy
  bpac init Assignment prod-subscription`,
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
metadata:
  name: %s
  environment: production
  owner: devops@company.com

spec:
  timezone: "Europe/Copenhagen"
  vault:
    name: backup-vault-prod
    resource_group: rg-backups

  retention:
    daily:
      count: 30
    weekly:
      day: Sunday
      count: 12
    monthly:
      count: 12
    yearly:
      count: 5

  schedule:
    full:
      frequency: daily
      time: "02:00"
    differential:
      enabled: true
    log:
      frequency_minutes: 60
`, name)
}

func assignmentTemplate(name string) string {
	return fmt.Sprintf(`apiVersion: backupctl/v1
kind: Assignment
metadata:
  name: %s
  subscriptionId: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  environment: production
  owner: devops@company.com

spec:
  assignments:
    - policy: gold-backup-policy
      targets:
        - type: azure_vm
          resource_group: rg-prod
          resource_name: vm-prod-web-01
`, name)
}
