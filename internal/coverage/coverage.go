package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Run is the main entry point for the coverage action.
func Run() error {
	configFile := getInput("CONFIG", "coverage.json")
	workDir := getInput("WORKING-DIRECTORY", ".")
	failOnError := getInput("FAIL-ON-ERROR", "true") == "true"

	configPath := filepath.Join(workDir, configFile)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	var allResults []EntryResult
	var allViolations []Violation
	allPassed := true

	for _, entry := range cfg.Coverage {
		reportPath := filepath.Join(workDir, entry.Path)
		data, err := os.ReadFile(reportPath)
		if err != nil {
			return fmt.Errorf("reading coverage file for %q: %w", entry.Name, err)
		}

		parser, err := getParser(entry.Format)
		if err != nil {
			return fmt.Errorf("entry %q: %w", entry.Name, err)
		}

		result, err := parser(data)
		if err != nil {
			return fmt.Errorf("parsing coverage for %q: %w", entry.Name, err)
		}
		result.Name = entry.Name

		cr := CheckThresholds(result, &entry.Threshold)

		entryResult := EntryResult{
			Name:   entry.Name,
			Passed: cr.Passed,
		}
		if result.Line != nil {
			pct := result.Line.Pct()
			entryResult.Line = &pct
		}
		if result.Branch != nil {
			pct := result.Branch.Pct()
			entryResult.Branch = &pct
		}
		if result.Function != nil {
			pct := result.Function.Pct()
			entryResult.Function = &pct
		}

		allResults = append(allResults, entryResult)

		if !cr.Passed {
			allPassed = false
			allViolations = append(allViolations, cr.Violations...)
		}

		for _, s := range cr.Skipped {
			EmitAnnotation("notice", fmt.Sprintf("%s: %s threshold configured but not reported by %s format — skipped",
				s.Entry, s.Metric, entry.Format))
		}
	}

	// Emit annotations
	for _, v := range allViolations {
		level := "error"
		if !failOnError {
			level = "warning"
		}
		EmitAnnotation(level, FormatViolation(v))
	}

	for _, r := range allResults {
		if r.Passed {
			parts := []string{}
			if r.Line != nil {
				parts = append(parts, fmt.Sprintf("line %.1f%%", *r.Line))
			}
			if r.Branch != nil {
				parts = append(parts, fmt.Sprintf("branch %.1f%%", *r.Branch))
			}
			if r.Function != nil {
				parts = append(parts, fmt.Sprintf("function %.1f%%", *r.Function))
			}
			msg := fmt.Sprintf("%s: %s — all thresholds met", r.Name, strings.Join(parts, ", "))
			EmitAnnotation("notice", msg)
		}
	}

	// Write job summary and outputs
	if err := WriteJobSummary(allResults); err != nil {
		EmitAnnotation("warning", fmt.Sprintf("failed to write job summary: %v", err))
	}

	if err := WriteOutputs(allPassed, allResults); err != nil {
		EmitAnnotation("warning", fmt.Sprintf("failed to write outputs: %v", err))
	}

	if !allPassed && failOnError {
		names := []string{}
		for _, v := range allViolations {
			names = append(names, v.Entry)
		}
		return fmt.Errorf("coverage below threshold for: %s", joinUnique(names))
	}

	return nil
}

func getInput(name, defaultVal string) string {
	val := os.Getenv("INPUT_" + name)
	if val == "" {
		return defaultVal
	}
	return val
}

func joinUnique(items []string) string {
	seen := map[string]bool{}
	unique := []string{}
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			unique = append(unique, item)
		}
	}
	return strings.Join(unique, ", ")
}
