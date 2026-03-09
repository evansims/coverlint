package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// defaultPaths maps coverage formats to common default output paths,
// ordered by likelihood. These are the standard output locations for
// the most popular coverage tools in each ecosystem.
var defaultPaths = map[string][]string{
	"lcov": {
		"coverage/lcov.info",
		"lcov.info",
		"coverage.lcov",
	},
	"gocover": {
		"cover.out",
		"coverage.out",
		"c.out",
	},
	"cobertura": {
		"coverage.xml",
		"cobertura.xml",
		"cobertura-coverage.xml",
	},
	"clover": {
		"coverage.xml",
		"clover.xml",
	},
	"jacoco": {
		"build/reports/jacoco/test/jacocoTestReport.xml",
		"target/site/jacoco/jacoco.xml",
		"jacoco.xml",
	},
}

// DiscoverReports searches for coverage report files using the default paths
// for the given format. It returns all paths that exist, relative to workDir.
func DiscoverReports(format, workDir string) ([]string, error) {
	paths, ok := defaultPaths[format]
	if !ok {
		return nil, fmt.Errorf("no default paths configured for format %q", format)
	}

	var found []string
	for _, p := range paths {
		full := filepath.Join(workDir, p)
		if _, err := os.Stat(full); err == nil {
			found = append(found, p)
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("auto-discovery: no %s coverage report found in %q (searched: %v)", format, workDir, paths)
	}

	return found, nil
}

// validatePathContainment checks that a resolved path stays within workDir.
func validatePathContainment(resolvedPath, workDir string) error {
	absWork, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolving working directory: %w", err)
	}
	absPath, err := filepath.Abs(filepath.Join(workDir, resolvedPath))
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	if !strings.HasPrefix(absPath, absWork+string(filepath.Separator)) && absPath != absWork {
		return fmt.Errorf("path %q escapes working directory", resolvedPath)
	}
	return nil
}

// ResolvePaths expands a path input (which may contain globs and/or
// comma-separated values) into a list of actual file paths relative
// to workDir. Returns an error if no files match or if any path
// escapes the working directory.
func ResolvePaths(pathInput, workDir string) ([]string, error) {
	var patterns []string
	for _, p := range strings.Split(pathInput, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			patterns = append(patterns, p)
		}
	}

	seen := map[string]bool{}
	var resolved []string

	for _, pattern := range patterns {
		fullPattern := filepath.Join(workDir, pattern)

		// Try glob expansion
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}

		if len(matches) > 0 {
			for _, match := range matches {
				// Convert back to relative path
				rel, err := filepath.Rel(workDir, match)
				if err != nil {
					rel = match
				}
				if !seen[rel] {
					seen[rel] = true
					resolved = append(resolved, rel)
				}
			}
		} else {
			// Not a glob — check if literal file exists
			if _, err := os.Stat(fullPattern); err == nil {
				if !seen[pattern] {
					seen[pattern] = true
					resolved = append(resolved, pattern)
				}
			} else {
				return nil, fmt.Errorf("coverage file not found: %q", fullPattern)
			}
		}
	}

	if len(resolved) == 0 {
		return nil, fmt.Errorf("no coverage files found matching %q", pathInput)
	}

	// Validate all resolved paths stay within workDir
	for _, p := range resolved {
		if err := validatePathContainment(p, workDir); err != nil {
			return nil, err
		}
	}

	return resolved, nil
}
