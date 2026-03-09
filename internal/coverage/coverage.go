package coverage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxCoverageFileSize is the maximum allowed size for a coverage report file (50 MB).
const maxCoverageFileSize = 50 * 1024 * 1024

// readCoverageFile reads a coverage file with size validation.
// Uses a single file handle to avoid TOCTOU between size check and read.
func readCoverageFile(path string) (data []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading coverage file %q: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing coverage file %q: %w", path, cerr)
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading coverage file %q: %w", path, err)
	}
	if info.Size() > maxCoverageFileSize {
		return nil, fmt.Errorf("coverage file %q exceeds maximum size of %d bytes (%d bytes)", path, maxCoverageFileSize, info.Size())
	}

	data, err = io.ReadAll(f)
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
		return &ConfigError{Message: err.Error(), Cause: err}
	}

	annotator := NewAnnotator(inp.Annotations, os.Stdout)

	// Parse coverage files grouped by format
	var perFormat []formatResult

	if strings.TrimSpace(inp.Path) == "" && inp.AutoFormat {
		// Both format and path auto-discovered
		discovered, err := DiscoverAllReports(inp.WorkDir)
		if err != nil {
			return &ConfigError{Message: err.Error(), Cause: err}
		}
		annotator.Emit("notice", fmt.Sprintf("auto-discovered %d report(s): %s",
			len(discovered), strings.Join(discovered, ", ")))

		perFormat, err = parseWithFormats(discovered, inp.Formats, inp.WorkDir)
		if err != nil {
			return &ConfigError{Message: err.Error(), Cause: err}
		}

		var detectedFormats []string
		for _, fr := range perFormat {
			detectedFormats = append(detectedFormats, fr.Format)
		}
		annotator.Emit("notice", fmt.Sprintf("auto-detected format(s): %s",
			strings.Join(detectedFormats, ", ")))
	} else if strings.TrimSpace(inp.Path) == "" {
		// Explicit formats, auto-discover paths per format
		for _, format := range inp.Formats {
			results, err := discoverAndParse(format, inp.WorkDir, annotator)
			if err != nil {
				return &ConfigError{Message: err.Error(), Cause: err}
			}
			perFormat = append(perFormat, formatResult{Format: format, Results: results})
		}
	} else {
		// Explicit paths: resolve once, then route each file to matching parser
		resolved, err := ResolvePaths(inp.Path, inp.WorkDir)
		if err != nil {
			return &ConfigError{Message: err.Error(), Cause: err}
		}
		perFormat, err = parseWithFormats(resolved, inp.Formats, inp.WorkDir)
		if err != nil {
			return &ConfigError{Message: err.Error(), Cause: err}
		}
	}

	if len(perFormat) == 0 {
		return &ConfigError{Message: "no coverage reports were parsed"}
	}

	// Build per-format merged results and entry results
	multiFormat := len(perFormat) > 1
	var allParsed []*CoverageResult
	var entryResults []EntryResult

	for _, fr := range perFormat {
		allParsed = append(allParsed, fr.Results...)

		if multiFormat {
			entry := buildEntryResult(fr.Format, MergeResults(fr.Results), inp.Threshold.Weights)
			entry.Passed = true // per-format rows don't show pass/fail
			entryResults = append(entryResults, entry)
		}
	}

	// Merge all results for the combined/total result
	combined := MergeResults(allParsed)
	if len(allParsed) > 1 {
		annotator.Emit("notice", fmt.Sprintf("merged %d coverage reports", len(allParsed)))
	}
	cr := CheckThresholds(combined, &inp.Threshold)
	hasThresholds := inp.Threshold.MinCoverage != nil || inp.Threshold.Line != nil || inp.Threshold.Branch != nil || inp.Threshold.Function != nil

	// Single-format: label the row with the format name; multi-format: "Total"
	var totalLabel string
	if multiFormat {
		totalLabel = "Total"
	} else {
		totalLabel = perFormat[0].Format
	}
	totalEntry := buildEntryResult(totalLabel, combined, inp.Threshold.Weights)
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

	// Collect actual format names for messages
	var formatNames []string
	for _, fr := range perFormat {
		formatNames = append(formatNames, fr.Format)
	}

	for _, s := range cr.Skipped {
		annotator.Emit("notice", fmt.Sprintf("%s: %s threshold configured but not reported by %s format — skipped",
			s.Entry, s.Metric, strings.Join(formatNames, ", ")))
	}

	// Emit annotations
	for _, v := range cr.Violations {
		level := "error"
		if !inp.FailOnError {
			level = "warning"
		}
		annotator.Emit(level, FormatViolation(v))
	}

	if cr.Passed {
		var parts []string
		if totalEntry.Score != nil {
			parts = append(parts, fmt.Sprintf("score %.1f%%", *totalEntry.Score))
		}
		if totalEntry.Line != nil {
			parts = append(parts, fmt.Sprintf("line %.1f%%", *totalEntry.Line))
		}
		if totalEntry.Branch != nil {
			parts = append(parts, fmt.Sprintf("branch %.1f%%", *totalEntry.Branch))
		}
		if totalEntry.Function != nil {
			parts = append(parts, fmt.Sprintf("function %.1f%%", *totalEntry.Function))
		}
		if hasThresholds && len(parts) > 0 {
			msg := fmt.Sprintf("coverage: %s — all minimums met", strings.Join(parts, ", "))
			annotator.Emit("notice", msg)
		} else if len(parts) > 0 {
			annotator.Emit("notice", fmt.Sprintf("coverage: %s", strings.Join(parts, ", ")))
		}
	}

	// Baseline comparison (before writing outputs so cr.Passed reflects delta violations)
	if inp.Baseline != "" {
		prev, loadErr := LoadBaseline(inp.Baseline)
		if loadErr != nil {
			annotator.Emit("warning", fmt.Sprintf("failed to load baseline: %v", loadErr))
		} else {
			deltaViolations := CompareBaseline(prev, cr.Score, inp.MinDelta)
			if len(deltaViolations) > 0 {
				cr.Violations = append(cr.Violations, deltaViolations...)
				cr.Passed = false
				totalEntry.Passed = cr.Passed
				for _, v := range deltaViolations {
					level := "error"
					if !inp.FailOnError {
						level = "warning"
					}
					annotator.Emit(level, FormatViolation(v))
				}
			}
		}
	} else if inp.MinDelta != nil {
		annotator.Emit("warning", "min-delta is set but no baseline provided — delta comparison skipped")
	}

	// Compute suggestions if enabled
	var suggestions []Suggestion
	if inp.Suggestions && len(combined.Files) > 0 {
		suggestions = RankSuggestions(combined.Files)
	}

	// Generate SARIF report if configured
	var sarifJSON string
	if inp.SARIF {
		sarifDoc := GenerateSARIF(combined.Files, combined.FileDetails, combined.BlockDetails)
		sarifBytes, merr := json.MarshalIndent(sarifDoc, "", "  ")
		if merr != nil {
			annotator.Emit("warning", fmt.Sprintf("failed to marshal SARIF: %v", merr))
		} else {
			sarifJSON = string(sarifBytes)
			annotator.Emit("notice", "SARIF output generated")
		}
	}

	// Write job summary and outputs
	// For multi-format, include per-format rows + total footer
	allResults := resultsForOutput
	if totalForSummary != nil {
		allResults = append(allResults, *totalForSummary)
	}

	if err := WriteJobSummary(allResults, totalForSummary != nil, suggestions); err != nil {
		annotator.Emit("warning", fmt.Sprintf("failed to write job summary: %v", err))
	}

	// Always generate baseline output
	var baselineOutput *BaselineData
	bd := GenerateBaseline(allResults)
	baselineOutput = &bd

	if err := WriteOutputs(cr.Passed, allResults, baselineOutput, sarifJSON); err != nil {
		annotator.Emit("warning", fmt.Sprintf("failed to write outputs: %v", err))
	}

	if !cr.Passed && inp.FailOnError {
		return &ThresholdError{Message: "coverage below threshold"}
	}

	return nil
}

// parseWithFormats tries each parser in order against each file and groups
// results by which parser succeeded. Used when paths are known but format
// may need to be auto-detected (or when multiple formats are configured
// with explicit paths).
func parseWithFormats(paths []string, formats []string, workDir string) ([]formatResult, error) {
	type namedParser struct {
		name   string
		parser parserFunc
	}
	var nps []namedParser
	for _, f := range formats {
		p, err := getParser(f)
		if err != nil {
			return nil, err
		}
		nps = append(nps, namedParser{name: f, parser: p})
	}

	results := map[string][]*CoverageResult{}
	for _, p := range paths {
		fullPath := filepath.Join(workDir, p)
		data, err := readCoverageFile(fullPath)
		if err != nil {
			return nil, err
		}

		matched := false
		for _, np := range nps {
			result, err := np.parser(data)
			if err == nil {
				results[np.name] = append(results[np.name], result)
				matched = true
				break
			}
		}
		if !matched {
			return nil, fmt.Errorf("parsing %q: no configured parser succeeded", p)
		}
	}

	// Preserve format order from input
	var out []formatResult
	for _, f := range formats {
		if r, ok := results[f]; ok {
			out = append(out, formatResult{Format: f, Results: r})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no coverage reports could be parsed")
	}
	return out, nil
}

// buildEntryResult creates an EntryResult from a CoverageResult.
func buildEntryResult(name string, r *CoverageResult, w Weights) EntryResult {
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
	score := CoverageScore(entry.Line, entry.Branch, entry.Function, w)
	entry.Score = &score
	return entry
}

// discoverAndParse auto-discovers and parses coverage files for a single format.
func discoverAndParse(format, workDir string, annotator *Annotator) ([]*CoverageResult, error) {
	parser, err := getParser(format)
	if err != nil {
		return nil, err
	}

	discovered, err := DiscoverReports(format, workDir)
	if err != nil {
		return nil, err
	}
	annotator.Emit("notice", fmt.Sprintf("auto-discovered %d %s report(s): %s",
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
