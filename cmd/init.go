package cmd

import (
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed templates
var templatesFS embed.FS

var validKinds = map[string]bool{
	"policy":     true,
	"assignment": true,
}

var validResourceTypes = map[string]bool{
	"azure_vm":     true,
	"azure_sql_db": true,
	"azure_disk":   true,
}

var initCmd = &cobra.Command{
	Use:   "init <kind> <resourceType> <out>",
	Short: "Scaffold a new Policy or Assignment YAML file",
	Long: `Generates a starter YAML file for the given kind (Policy or Assignment)
and resource type (azure_vm, azure_sql_db, azure_disk).

Examples:
  bpac init policy azure_vm gold-vm-daily.yaml
  bpac init assignment azure_vm prod-vm-assignments.yaml`,
	Args: cobra.ExactArgs(3),
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	kind := strings.ToLower(args[0])
	resourceType := strings.ToLower(args[1])
	out := args[2]

	content, err := loadTemplate(kind, resourceType)
	if err != nil {
		return err
	}

	if err := writeFile(content, out); err != nil {
		return err
	}

	return nil
}

// loadTemplate validates kind/resourceType and returns the matching embedded
// template's contents.
func loadTemplate(kind string, resourceType string) (string, error) {
	if !validKinds[kind] {
		return "", fmt.Errorf("invalid kind %q: must be one of policy, assignment", kind)
	}

	if !validResourceTypes[resourceType] {
		return "", fmt.Errorf("invalid resource type %q: must be one of azure_vm, azure_sql_db, azure_disk", resourceType)
	}

	tmplPath := path.Join("templates", kind, resourceType+".yml")

	data, err := templatesFS.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("no template found for kind %q and resource type %q: %w", kind, resourceType, err)
	}

	return string(data), nil
}

func writeFile(content string, out string) error {
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("file %q already exists", out)
	}

	if err := os.WriteFile(out, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %q: %w", out, err)
	}

	abs, err := filepath.Abs(out)
	if err != nil {
		abs = out
	}
	fmt.Printf("Created %s\n", abs)

	return nil
}
