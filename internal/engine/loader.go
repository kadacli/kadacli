package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/dbjorklund/backup-policy-as-code/internal/models"
)

// LoadAll walks dir recursively, detects each YAML file's kind, and returns
// separate slices of Policy and Assignment along with any parse errors.
func LoadAll(dir string) ([]models.PathPolicy, []models.PathAssignment, []error) {
	var errs []error
	var policies []models.PathPolicy
	var assignments []models.PathAssignment

	paths, err := findYAMLFiles(dir)
	if err != nil {
		return nil, nil, []error{err}
	}

	for _, path := range paths {
		kind, err := detectKind(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}

		switch kind {
		case "Policy":
			p, err := loadPolicy(path)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", path, err))
			} else {
				policies = append(policies, models.PathPolicy{Path: path, Policy: p})
			}
		case "Assignment":
			a, err := loadAssignment(path)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", path, err))
			} else {
				assignments = append(assignments, models.PathAssignment{Path: path, Assignment: a})
			}
		default:
			errs = append(errs, fmt.Errorf("%s: unknown kind %q (expected Policy or Assignment)", path, kind))
		}
	}

	return policies, assignments, errs
}

func findYAMLFiles(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_")) {
			return filepath.SkipDir
		}
		if !d.IsDir() && isYAML(path) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// detectKind reads only the top-level "kind" field from a YAML file.
func detectKind(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var top struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(data, &top); err != nil {
		return "", err
	}
	if top.Kind == "" {
		return "", fmt.Errorf("missing required field: kind")
	}
	return top.Kind, nil
}

func loadPolicy(path string) (models.Policy, error) {
	var p models.Policy
	data, err := os.ReadFile(path)
	if err != nil {
		return p, err
	}
	if err := yaml.Unmarshal(data, &p); err != nil {
		return p, err
	}
	return p, nil
}

func loadAssignment(path string) (models.Assignment, error) {
	var a models.Assignment
	data, err := os.ReadFile(path)
	if err != nil {
		return a, err
	}
	if err := yaml.Unmarshal(data, &a); err != nil {
		return a, err
	}
	return a, nil
}

func isYAML(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}
