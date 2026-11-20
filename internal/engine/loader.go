package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/kadacli/kadacli/internal/models"
)

// LoadResult reports what a scan of a directory turned up. Skipped counts the
// YAML files that are not kåda documents at all — a policy directory inside a
// wider repo will contain plenty of those, and they are not errors.
type LoadResult struct {
	Policies    []models.PathPolicy
	Assignments []models.PathAssignment
	Skipped     []string
	Errs        []error
}

// LoadAll walks dir recursively, detects each YAML file's kind, and returns
// separate slices of Policy and Assignment along with any parse errors.
//
// Files that are not kåda documents — no `kind` field, an unrecognised `kind`,
// or not parseable as YAML at all — are skipped rather than failing the run: a
// policy directory frequently sits alongside compose files, Helm templates and
// CI config, and one of those must not stop the tool from reading the policies
// next to it. Skipped paths are reported so a typo'd `kind` is still visible.
func LoadAll(dir string) LoadResult {
	var result LoadResult

	paths, err := findYAMLFiles(dir)
	if err != nil {
		result.Errs = []error{err}
		return result
	}

	for _, path := range paths {
		kind, err := detectKind(path)
		if err != nil {
			// Unreadable or unparseable: it cannot be one of ours.
			result.Skipped = append(result.Skipped, path)
			continue
		}

		switch kind {
		case "Policy":
			p, err := loadPolicy(path)
			if err != nil {
				result.Errs = append(result.Errs, fmt.Errorf("%s: %w", path, err))
			} else {
				result.Policies = append(result.Policies, models.PathPolicy{Path: path, Policy: p})
			}
		case "Assignment":
			a, err := loadAssignment(path)
			if err != nil {
				result.Errs = append(result.Errs, fmt.Errorf("%s: %w", path, err))
			} else {
				result.Assignments = append(result.Assignments, models.PathAssignment{Path: path, Assignment: a})
			}
		default:
			result.Skipped = append(result.Skipped, path)
		}
	}

	return result
}

func findYAMLFiles(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// The walk root is whatever the user asked for and is never
			// skipped. WalkDir reports it with d.Name() == "." for --dir .,
			// which the hidden-directory rule below would otherwise match —
			// skipping the root ends the walk immediately and the whole run
			// finds zero files while reporting success.
			if path == dir {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if isYAML(path) {
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
		return "", fmt.Errorf("Missing required field: kind")
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
