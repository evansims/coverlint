package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxCoverageFileSize is the maximum allowed size for a coverage report file (50 MB).
const maxCoverageFileSize = 50 * 1024 * 1024

// readCoverageFile reads a coverage file with size validation.
func readCoverageFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading coverage file %q: %w", path, err)
	}
	if info.Size() > maxCoverageFileSize {
		return nil, fmt.Errorf("coverage file %q exceeds maximum size of %d bytes (%d bytes)", path, maxCoverageFileSize, info.Size())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading coverage file %q: %w", path, err)
	}
	return data, nil
}

// formatResult pairs a format name with its parsed coverage results.
type formatResult struct {
	Format  string
	Results []*CoverageResult
}

// Run is the main entry point for the coverage action.
func Run() error {
	inp, err := ParseInputs()
	if err != nil {
		return err
	}

	// Parse coverage files grouped by format
	var perFormat []formatResult

	if strings.TrimSpace(inp.Path) == "" {
		// Auto-discovery: each format discovers its own files independently
		for _, format := range inp.Formats {
			results, err := discoverAndParse(format, inp.WorkDir)
			if err != nil {
				return err
			}
			perFormat = append(perFormat, formatResult{Format: format, Results: results})
		}
	} else {
		// Explicit paths: resolve once, then route each file to matching parser
		resolved, err := ResolvePaths(inp.Path, inp.WorkDir)
		if err != nil {
			return err
		}

		type namedParser struct {
			name   string
			parser parserFunc
		}
		var namedParsers []namedParser
		for _, f := range inp.Formats {
			p, err := getParser(f)
			if err != nil {
				return err
			}
			namedParsers = append(namedParsers, namedParser{name: f, parser: p})
		}

		// Group results by which parser succeeded
		formatResults := map[string][]*CoverageResult{}
		for _, p := range resolved {
			fullPath := filepath.Join(inp.WorkDir, p)
			data, err := readCoverageFile(fullPath)
			if err != nil {
				return err
			}

			matched := false
			for _, np := range namedParsers {
				result, err := np.parser(data)
				if err == nil {
					formatResults[np.name] = append(formatResults[np.name], result)
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf("parsing %q: no configured parser succeeded", p)
			}
		}

		// Preserve format order from input
		for _, f := range inp.Formats {
			if results, ok := formatResults[f]; ok {
				perFormat = append(perFormat, formatResult{Format: f, Results: results})
			}
		}
	}

	// Build per-format merged results and entry results
	multiFormat := len(inp.Formats) > 1
	var allParsed []*CoverageResult
	var entryResults []EntryResult

	for _, fr := range perFormat {
		allParsed = append(allParsed, fr.Results...)

		if multiFormat {
			// Merge within this format if multiple files
			var formatMerged *CoverageResult
			if len(fr.Results) == 1 {
				formatMerged = fr.Results[0]
			} else {
				formatMerged = MergeResults(fr.Results)
			}

			entry := buildEntryResult(fr.Format, formatMerged)
			entry.Passed = true // per-format rows don't show pass/fail
			entryResults = append(entryResults, entry)
		}
	}

	// Merge all results for the combined/total result
	var combined *CoverageResult
	if len(allParsed) == 1 {
		combined = allParsed[0]
	} else {
		combined = MergeResults(allParsed)
		EmitAnnotation("notice", fmt.Sprintf("merged %d coverage reports", len(allParsed)))
	}
	cr := CheckThresholds(combined, &inp.Threshold)

	// Single-format: label the row with the format name; multi-format: "Total"
	var totalLabel string
	if multiFormat {
		totalLabel = "Total"
	} else {
		totalLabel = inp.Formats[0]
	}
	totalEntry := buildEntryResult(totalLabel, combined)
	totalEntry.Passed = cr.Passed

	// For single-format, the results list is just the total entry
	// For multi-format, per-format rows are in entryResults and total is separate
	var resultsForOutput []EntryResult
	var totalForSummary *EntryResult
	if multiFormat {
		resultsForOutput = entryResults
		totalForSummary = &totalEntry
	} else {
		resultsForOutput = []EntryResult{totalEntry}
	}

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
		if totalEntry.Line != nil {
			parts = append(parts, fmt.Sprintf("line %.1f%%", *totalEntry.Line))
		}
		if totalEntry.Branch != nil {
			parts = append(parts, fmt.Sprintf("branch %.1f%%", *totalEntry.Branch))
		}
		if totalEntry.Function != nil {
			parts = append(parts, fmt.Sprintf("function %.1f%%", *totalEntry.Function))
		}
		msg := fmt.Sprintf("coverage: %s — all thresholds met", strings.Join(parts, ", "))
		EmitAnnotation("notice", msg)
	}

	// Compute suggestions if enabled
	var suggestions []Suggestion
	if inp.Suggestions && len(combined.Files) > 0 {
		suggestions = RankSuggestions(combined.Files)
	}

	// Write job summary and outputs
	// For multi-format, include per-format rows + total footer
	allResults := resultsForOutput
	if totalForSummary != nil {
		allResults = append(allResults, *totalForSummary)
	}

	if err := WriteJobSummary(allResults, totalForSummary != nil, suggestions); err != nil {
		EmitAnnotation("warning", fmt.Sprintf("failed to write job summary: %v", err))
	}

	if err := WriteOutputs(cr.Passed, allResults); err != nil {
		EmitAnnotation("warning", fmt.Sprintf("failed to write outputs: %v", err))
	}

	if !cr.Passed && inp.FailOnError {
		return fmt.Errorf("coverage below threshold")
	}

	return nil
}

// buildEntryResult creates an EntryResult from a CoverageResult.
func buildEntryResult(name string, r *CoverageResult) EntryResult {
	entry := EntryResult{Name: name}
	if r.Line != nil {
		pct := r.Line.Pct()
		entry.Line = &pct
	}
	if r.Branch != nil {
		pct := r.Branch.Pct()
		entry.Branch = &pct
	}
	if r.Function != nil {
		pct := r.Function.Pct()
		entry.Function = &pct
	}
	return entry
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
		data, err := readCoverageFile(fullPath)
		if err != nil {
			return nil, err
		}

		result, err := parser(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %q: %w", p, err)
		}
		results = append(results, result)
	}

	return results, nil
}
