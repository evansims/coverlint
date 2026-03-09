package coverage

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Input holds the parsed and validated action inputs.
type Input struct {
	Name        string
	Path        string
	Formats     []string
	WorkDir     string
	FailOnError bool
	Suggestions bool
	Threshold   Threshold
}

var validFormats = map[string]bool{
	"lcov":      true,
	"gocover":   true,
	"cobertura": true,
	"clover":    true,
	"jacoco":    true,
}

// ParseInputs reads action inputs from INPUT_* environment variables and validates them.
func ParseInputs() (*Input, error) {
	inp := &Input{
		Name:        getInput("NAME", ""),
		Path:        getInput("PATH", ""),
		WorkDir:     getInput("WORKING-DIRECTORY", "."),
		FailOnError: getInput("FAIL-ON-ERROR", "true") == "true",
		Suggestions: getInput("SUGGESTIONS", "true") == "true",
	}

	formatRaw := getInput("FORMAT", "")
	if strings.TrimSpace(formatRaw) == "" {
		return nil, fmt.Errorf("input validation: format is required")
	}

	for _, f := range strings.Split(formatRaw, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !validFormats[f] {
			return nil, fmt.Errorf("input validation: format %q is not valid (valid: lcov, gocover, cobertura, clover, jacoco)", f)
		}
		inp.Formats = append(inp.Formats, f)
	}

	if len(inp.Formats) == 0 {
		return nil, fmt.Errorf("input validation: format is required")
	}

	if inp.Name == "" {
		inp.Name = strings.Join(inp.Formats, ", ")
	}

	// Sanitize name to prevent injection via newlines or control characters
	inp.Name = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, inp.Name)

	line, err := parseOptionalFloat(os.Getenv("INPUT_THRESHOLD-LINE"))
	if err != nil {
		return nil, fmt.Errorf("input validation: threshold-line: %w", err)
	}
	branch, err := parseOptionalFloat(os.Getenv("INPUT_THRESHOLD-BRANCH"))
	if err != nil {
		return nil, fmt.Errorf("input validation: threshold-branch: %w", err)
	}
	function, err := parseOptionalFloat(os.Getenv("INPUT_THRESHOLD-FUNCTION"))
	if err != nil {
		return nil, fmt.Errorf("input validation: threshold-function: %w", err)
	}

	if line == nil && branch == nil && function == nil {
		return nil, fmt.Errorf("input validation: at least one threshold (threshold-line, threshold-branch, threshold-function) is required")
	}

	inp.Threshold = Threshold{Line: line, Branch: branch, Function: function}
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
