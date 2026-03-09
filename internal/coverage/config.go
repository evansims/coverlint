package coverage

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// formatOrder defines the priority order for format auto-detection.
// Most distinctive/unambiguous formats first to minimize false matches.
// This must stay in sync with the parsers map in parser.go.
var formatOrder = []string{"gocover", "lcov", "jacoco", "cobertura", "clover"}

func init() {
	// Enforce that formatOrder and parsers stay in sync.
	if len(formatOrder) != len(parsers) {
		panic("formatOrder and parsers are out of sync")
	}
	for _, f := range formatOrder {
		if _, ok := parsers[f]; !ok {
			panic("formatOrder contains unknown format: " + f)
		}
	}
}

// AnnotationConfig controls how annotations are emitted.
type AnnotationConfig struct {
	Mode     string // "all", "none", or "limited"
	MaxCount int    // only used when Mode is "limited"
}

// Input holds the parsed and validated action inputs.
type Input struct {
	Path        string
	Formats     []string
	AutoFormat  bool
	WorkDir     string
	FailOnError bool
	Suggestions bool
	Annotations AnnotationConfig
	Threshold   Threshold
}

// ParseInputs reads action inputs from INPUT_* environment variables and validates them.
func ParseInputs() (*Input, error) {
	inp := &Input{
		Path:        getInput("PATH", ""),
		WorkDir:     getInput("WORKING-DIRECTORY", "."),
		FailOnError: getInput("FAIL-ON-ERROR", "true") == "true",
		Suggestions: getInput("SUGGESTIONS", "true") == "true",
	}

	formats := splitList(getInput("FORMAT", ""))
	if len(formats) == 0 {
		inp.AutoFormat = true
		inp.Formats = formatOrder
	} else {
		for _, f := range formats {
			if _, ok := parsers[f]; !ok {
				return nil, fmt.Errorf("input validation: format %q is not valid (valid: lcov, gocover, cobertura, clover, jacoco)", f)
			}
			inp.Formats = append(inp.Formats, f)
		}
	}

	// Parse annotations config
	annotations := getInput("ANNOTATIONS", "true")
	switch annotations {
	case "true":
		inp.Annotations = AnnotationConfig{Mode: "all"}
	case "false":
		inp.Annotations = AnnotationConfig{Mode: "none"}
	default:
		n, err := strconv.Atoi(annotations)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("input validation: annotations must be 'true', 'false', or a non-negative integer, got %q", annotations)
		}
		inp.Annotations = AnnotationConfig{Mode: "limited", MaxCount: n}
	}

	// Parse min-coverage (weighted score threshold) and per-metric overrides
	minCoverage, err := parseOptionalFloat(os.Getenv("INPUT_MIN-COVERAGE"))
	if err != nil {
		return nil, fmt.Errorf("input validation: min-coverage: %w", err)
	}
	line, err := parseOptionalFloat(os.Getenv("INPUT_MIN-LINE"))
	if err != nil {
		return nil, fmt.Errorf("input validation: min-line: %w", err)
	}
	branch, err := parseOptionalFloat(os.Getenv("INPUT_MIN-BRANCH"))
	if err != nil {
		return nil, fmt.Errorf("input validation: min-branch: %w", err)
	}
	function, err := parseOptionalFloat(os.Getenv("INPUT_MIN-FUNCTION"))
	if err != nil {
		return nil, fmt.Errorf("input validation: min-function: %w", err)
	}

	// Parse weights (non-negative, defaults applied if not set)
	weights := DefaultWeights()
	if wl, err := parseOptionalFloat(os.Getenv("INPUT_WEIGHT-LINE")); err != nil {
		return nil, fmt.Errorf("input validation: weight-line: %w", err)
	} else if wl != nil {
		weights.Line = *wl
	}
	if wb, err := parseOptionalFloat(os.Getenv("INPUT_WEIGHT-BRANCH")); err != nil {
		return nil, fmt.Errorf("input validation: weight-branch: %w", err)
	} else if wb != nil {
		weights.Branch = *wb
	}
	if wf, err := parseOptionalFloat(os.Getenv("INPUT_WEIGHT-FUNCTION")); err != nil {
		return nil, fmt.Errorf("input validation: weight-function: %w", err)
	} else if wf != nil {
		weights.Function = *wf
	}

	inp.Threshold = Threshold{
		Line:        line,
		Branch:      branch,
		Function:    function,
		MinCoverage: minCoverage,
		Weights:     weights,
	}
	return inp, nil
}

func parseOptionalFloat(s string) (*float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid number", s)
	}
	if v < 0 || v > 100 {
		return nil, fmt.Errorf("%.1f must be between 0 and 100", v)
	}
	return &v, nil
}

func getInput(name, defaultVal string) string {
	val := os.Getenv("INPUT_" + name)
	if val == "" {
		return defaultVal
	}
	return val
}

// splitList splits a string on commas and newlines, trims whitespace,
// and drops empty entries. Supports both comma-separated and YAML
// multiline (|) input styles.
func splitList(s string) []string {
	// Normalize newlines to commas so a single split handles both
	s = strings.ReplaceAll(s, "\n", ",")
	s = strings.ReplaceAll(s, "\r", ",")
	var out []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
