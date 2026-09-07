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

// Without this directive templatesFS is an empty filesystem and every init
// invocation fails with "no template found", regardless of resource type.
//
//go:embed templates/policy/*.yml templates/assignment/*.yml
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
and resource type.

Examples:
  kada init policy azure_vm gold-vm-daily.yaml
  kada init assignment azure_vm prod-vm-assignments.yaml`,
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
		return "", fmt.Errorf("Invalid kind %q: must be one of policy, assignment", kind)
	}

	if !validResourceTypes[resourceType] {
		return "", fmt.Errorf("Invalid resource type %q: must be one of azure_vm, azure_sql_db, azure_disk", resourceType)
	}

	tmplPath := path.Join("templates", kind, resourceType+".yml")

	data, err := templatesFS.ReadFile(tmplPath)
	if err != nil {
		return "", fmt.Errorf("No template found for kind %q and resource type %q: %w", kind, resourceType, err)
	}

	return string(data), nil
}

func writeFile(content string, out string) error {
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("File %q already exists", out)
	}

	if err := os.WriteFile(out, []byte(content), 0644); err != nil {
		return fmt.Errorf("Failed to write %q: %w", out, err)
	}

	abs, err := filepath.Abs(out)
	if err != nil {
		abs = out
	}
	fmt.Println(colorKada("Created " + abs))

	return nil
}
