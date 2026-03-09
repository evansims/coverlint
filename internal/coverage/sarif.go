package coverage

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// maxSARIFResults caps the number of results in a SARIF document to keep output manageable.
const maxSARIFResults = 1000

// SARIFDocument represents a SARIF 2.1.0 log file.
type SARIFDocument struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

// SARIFRun represents a single analysis run.
type SARIFRun struct {
	Tool    SARIFTool     `json:"tool"`
	Results []SARIFResult `json:"results"`
}

// SARIFTool describes the analysis tool.
type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

// SARIFDriver describes the tool driver (name, version, rules).
type SARIFDriver struct {
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Rules   []SARIFRule `json:"rules"`
}

// SARIFRule defines a reportable rule.
type SARIFRule struct {
	ID               string          `json:"id"`
	ShortDescription SARIFMessage    `json:"shortDescription"`
	DefaultConfig    SARIFRuleConfig `json:"defaultConfiguration"`
}

// SARIFRuleConfig holds the default severity level for a rule.
type SARIFRuleConfig struct {
	Level string `json:"level"`
}

// SARIFResult represents a single finding.
type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations"`
}

// SARIFMessage holds a human-readable message.
type SARIFMessage struct {
	Text string `json:"text"`
}

// SARIFLocation describes where a result was found.
type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

// SARIFPhysicalLocation identifies a file and optional region.
type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
	Region           *SARIFRegion          `json:"region,omitempty"`
}

// SARIFArtifactLocation identifies an artifact by URI.
type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// SARIFRegion identifies a region within a file.
type SARIFRegion struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine,omitempty"`
}

// GenerateSARIF creates a SARIF 2.1.0 document from coverage data.
// For line-based formats (fileDetails != nil), it emits one result per uncovered line.
// For block-based formats (blockDetails != nil), it emits one result per uncovered block.
// Results are capped at maxSARIFResults and sorted for deterministic output.
func GenerateSARIF(files []FileCoverage, fileDetails map[string]*FileLineDetail, blockDetails map[string]map[string]*BlockEntry) SARIFDocument {
	rules := []SARIFRule{
		{
			ID:               "coverage/uncovered-line",
			ShortDescription: SARIFMessage{Text: "Line not covered by tests"},
			DefaultConfig:    SARIFRuleConfig{Level: "note"},
		},
		{
			ID:               "coverage/uncovered-block",
			ShortDescription: SARIFMessage{Text: "Block not covered by tests"},
			DefaultConfig:    SARIFRuleConfig{Level: "note"},
		},
	}

	var results []SARIFResult

	if fileDetails != nil {
		results = generateLineResults(fileDetails)
	} else if blockDetails != nil {
		results = generateBlockResults(blockDetails)
	}

	// Cap results
	if len(results) > maxSARIFResults {
		results = results[:maxSARIFResults]
	}

	return SARIFDocument{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:    "coverlint",
						Version: "1.0.0",
						Rules:   rules,
					},
				},
				Results: results,
			},
		},
	}
}

// generateLineResults produces SARIF results for uncovered lines from line-based formats.
func generateLineResults(fileDetails map[string]*FileLineDetail) []SARIFResult {
	// Collect all file paths and sort for deterministic order
	paths := make([]string, 0, len(fileDetails))
	for p := range fileDetails {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var results []SARIFResult
	for _, path := range paths {
		detail := fileDetails[path]
		if detail.Lines == nil {
			continue
		}

		// Collect uncovered line numbers and sort
		var uncovered []int
		for line, count := range detail.Lines {
			if count == 0 {
				uncovered = append(uncovered, line)
			}
		}
		sort.Ints(uncovered)

		for _, line := range uncovered {
			results = append(results, SARIFResult{
				RuleID:  "coverage/uncovered-line",
				Message: SARIFMessage{Text: fmt.Sprintf("Line %d is not covered by tests", line)},
				Locations: []SARIFLocation{
					{
						PhysicalLocation: SARIFPhysicalLocation{
							ArtifactLocation: SARIFArtifactLocation{URI: path},
							Region:           &SARIFRegion{StartLine: line},
						},
					},
				},
			})
		}
	}

	return results
}

// generateBlockResults produces SARIF results for uncovered blocks from gocover format.
func generateBlockResults(blockDetails map[string]map[string]*BlockEntry) []SARIFResult {
	// Collect all file paths and sort for deterministic order
	paths := make([]string, 0, len(blockDetails))
	for p := range blockDetails {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	type blockResult struct {
		path      string
		startLine int
		endLine   int
		key       string
	}

	var uncovered []blockResult
	for _, path := range paths {
		blocks := blockDetails[path]
		// Collect and sort block keys for this file
		keys := make([]string, 0, len(blocks))
		for k := range blocks {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			entry := blocks[key]
			if entry.Count == 0 {
				start, end, err := parseBlockRange(key)
				if err != nil {
					continue
				}
				uncovered = append(uncovered, blockResult{
					path:      path,
					startLine: start,
					endLine:   end,
					key:       key,
				})
			}
		}
	}

	// Sort by path, then by start line
	sort.Slice(uncovered, func(i, j int) bool {
		if uncovered[i].path != uncovered[j].path {
			return uncovered[i].path < uncovered[j].path
		}
		return uncovered[i].startLine < uncovered[j].startLine
	})

	results := make([]SARIFResult, 0, len(uncovered))
	for _, u := range uncovered {
		region := &SARIFRegion{StartLine: u.startLine}
		if u.endLine != u.startLine {
			region.EndLine = u.endLine
		}
		results = append(results, SARIFResult{
			RuleID:  "coverage/uncovered-block",
			Message: SARIFMessage{Text: fmt.Sprintf("Block at lines %d-%d is not covered by tests", u.startLine, u.endLine)},
			Locations: []SARIFLocation{
				{
					PhysicalLocation: SARIFPhysicalLocation{
						ArtifactLocation: SARIFArtifactLocation{URI: u.path},
						Region:           region,
					},
				},
			},
		})
	}

	return results
}

// parseBlockRange parses a gocover block key like "file.go:5.1,10.1" and returns
// startLine=5 and endLine=10.
func parseBlockRange(key string) (startLine, endLine int, err error) {
	// Find the last ":" separator
	lastColon := strings.LastIndex(key, ":")
	if lastColon < 0 {
		return 0, 0, fmt.Errorf("no colon in block key %q", key)
	}
	rangeStr := key[lastColon+1:]

	parts := strings.SplitN(rangeStr, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("no comma in range %q", rangeStr)
	}

	startParts := strings.SplitN(parts[0], ".", 2)
	if len(startParts) < 1 {
		return 0, 0, fmt.Errorf("invalid start in range %q", rangeStr)
	}
	startLine, err = strconv.Atoi(startParts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing start line in %q: %w", rangeStr, err)
	}

	endParts := strings.SplitN(parts[1], ".", 2)
	if len(endParts) < 1 {
		return 0, 0, fmt.Errorf("invalid end in range %q", rangeStr)
	}
	endLine, err = strconv.Atoi(endParts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing end line in %q: %w", rangeStr, err)
	}

	return startLine, endLine, nil
}

// WriteSARIFFile marshals a SARIF document to JSON and writes it to the given path.
func WriteSARIFFile(doc SARIFDocument, path string) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling SARIF: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing SARIF file: %w", err)
	}

	return nil
}
