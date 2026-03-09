package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Run is the main entry point for the coverage action.
func Run() error {
	inp, err := ParseInputs()
	if err != nil {
		return err
	}

	parser, err := getParser(inp.Format)
	if err != nil {
		return err
	}

	// Resolve coverage report paths
	var reportPaths []string
	if strings.TrimSpace(inp.Path) == "" {
		// Auto-discover all matching reports
		discovered, err := DiscoverReports(inp.Format, inp.WorkDir)
		if err != nil {
			return err
		}
		reportPaths = discovered
		EmitAnnotation("notice", fmt.Sprintf("auto-discovered %d %s report(s): %s",
			len(discovered), inp.Format, strings.Join(discovered, ", ")))
	} else {
		// Resolve explicit path (supports globs and comma-separated)
		resolved, err := ResolvePaths(inp.Path, inp.WorkDir)
		if err != nil {
			return err
		}
		reportPaths = resolved
	}

	// Parse all report files
	var parsed []*CoverageResult
	for _, p := range reportPaths {
		fullPath := filepath.Join(inp.WorkDir, p)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("reading coverage file %q: %w", fullPath, err)
		}

		result, err := parser(data)
		if err != nil {
			return fmt.Errorf("parsing %q: %w", p, err)
		}
		parsed = append(parsed, result)
	}

	// Merge if multiple reports
	var result *CoverageResult
	if len(parsed) == 1 {
		result = parsed[0]
	} else {
		result = MergeResults(parsed)
		EmitAnnotation("notice", fmt.Sprintf("merged %d coverage reports", len(parsed)))
	}
	result.Name = inp.Name

	cr := CheckThresholds(result, &inp.Threshold)

	entryResult := EntryResult{
		Name:   inp.Name,
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

	results := []EntryResult{entryResult}

	for _, s := range cr.Skipped {
		EmitAnnotation("notice", fmt.Sprintf("%s: %s threshold configured but not reported by %s format — skipped",
			s.Entry, s.Metric, inp.Format))
	}

	// Emit annotations
	for _, v := range cr.Violations {
		level := "error"
		if !inp.FailOnError {
			level = "warning"
		}
		EmitAnnotation(level, FormatViolation(v))
	}

	if cr.Passed {
		var parts []string
		if entryResult.Line != nil {
			parts = append(parts, fmt.Sprintf("line %.1f%%", *entryResult.Line))
		}
		if entryResult.Branch != nil {
			parts = append(parts, fmt.Sprintf("branch %.1f%%", *entryResult.Branch))
		}
		if entryResult.Function != nil {
			parts = append(parts, fmt.Sprintf("function %.1f%%", *entryResult.Function))
		}
		msg := fmt.Sprintf("%s: %s — all thresholds met", inp.Name, strings.Join(parts, ", "))
		EmitAnnotation("notice", msg)
	}

	// Compute suggestions if enabled
	var suggestions []Suggestion
	if inp.Suggestions && len(result.Files) > 0 {
		suggestions = RankSuggestions(result.Files)
	}

	// Write job summary and outputs
	if err := WriteJobSummary(results, suggestions); err != nil {
		EmitAnnotation("warning", fmt.Sprintf("failed to write job summary: %v", err))
	}

	if err := WriteOutputs(cr.Passed, results); err != nil {
		EmitAnnotation("warning", fmt.Sprintf("failed to write outputs: %v", err))
	}

	if !cr.Passed && inp.FailOnError {
		return fmt.Errorf("coverage below threshold for: %s", inp.Name)
	}

	return nil
}
