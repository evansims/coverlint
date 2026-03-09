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

	var parsed []*CoverageResult

	if strings.TrimSpace(inp.Path) == "" {
		// Auto-discovery: each format discovers its own files independently
		for _, format := range inp.Formats {
			results, err := discoverAndParse(format, inp.WorkDir)
			if err != nil {
				return err
			}
			parsed = append(parsed, results...)
		}
	} else {
		// Explicit paths: resolve once, then route each file to matching parser
		resolved, err := ResolvePaths(inp.Path, inp.WorkDir)
		if err != nil {
			return err
		}

		parsersForFormats := make([]parserFunc, len(inp.Formats))
		for i, f := range inp.Formats {
			p, err := getParser(f)
			if err != nil {
				return err
			}
			parsersForFormats[i] = p
		}

		for _, p := range resolved {
			fullPath := filepath.Join(inp.WorkDir, p)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				return fmt.Errorf("reading coverage file %q: %w", fullPath, err)
			}

			result, err := tryParsers(data, p, parsersForFormats)
			if err != nil {
				return err
			}
			parsed = append(parsed, result)
		}
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
			s.Entry, s.Metric, strings.Join(inp.Formats, ", ")))
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

// discoverAndParse auto-discovers and parses coverage files for a single format.
func discoverAndParse(format, workDir string) ([]*CoverageResult, error) {
	parser, err := getParser(format)
	if err != nil {
		return nil, err
	}

	discovered, err := DiscoverReports(format, workDir)
	if err != nil {
		return nil, err
	}
	EmitAnnotation("notice", fmt.Sprintf("auto-discovered %d %s report(s): %s",
		len(discovered), format, strings.Join(discovered, ", ")))

	var results []*CoverageResult
	for _, p := range discovered {
		fullPath := filepath.Join(workDir, p)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("reading coverage file %q: %w", fullPath, err)
		}

		result, err := parser(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", p, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// tryParsers attempts to parse data with each parser in order, returning the
// first successful result. Used for multi-format explicit paths where each
// file's format is determined by which parser can successfully parse it.
func tryParsers(data []byte, path string, parsers []parserFunc) (*CoverageResult, error) {
	if len(parsers) == 1 {
		result, err := parsers[0](data)
		if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", path, err)
		}
		return result, nil
	}

	var lastErr error
	for _, parser := range parsers {
		result, err := parser(data)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("parsing %q: no parser succeeded (last error: %w)", path, lastErr)
}
