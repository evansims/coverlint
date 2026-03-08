package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatViolation(t *testing.T) {
	v := Violation{
		Entry:    "backend",
		Metric:   "line",
		Actual:   73.2,
		Required: 80,
	}
	msg := FormatViolation(v)
	if !strings.Contains(msg, "backend") {
		t.Errorf("message should contain entry name, got: %s", msg)
	}
	if !strings.Contains(msg, "73.2%") {
		t.Errorf("message should contain actual pct, got: %s", msg)
	}
	if !strings.Contains(msg, "80.0%") {
		t.Errorf("message should contain required pct, got: %s", msg)
	}
}

func TestWriteJobSummary(t *testing.T) {
	summaryFile := filepath.Join(t.TempDir(), "summary.md")
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	line1 := 87.3
	branch1 := 72.1
	func1 := 91.0
	line2 := 65.0
	branch2 := 55.0

	results := []EntryResult{
		{
			Name:     "backend",
			Line:     &line1,
			Branch:   &branch1,
			Function: &func1,
			Passed:   true,
		},
		{
			Name:     "frontend",
			Line:     &line2,
			Branch:   &branch2,
			Function: nil,
			Passed:   false,
		},
	}

	if err := WriteJobSummary(results); err != nil {
		t.Fatalf("WriteJobSummary() error: %v", err)
	}

	data, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	checks := []string{"backend", "frontend", "87.3%", "65.0%", "N/A", "Pass", "**Fail**"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("summary should contain %q", check)
		}
	}
}

func TestWriteOutputs(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)

	line := 87.3
	results := []EntryResult{
		{Name: "backend", Line: &line, Passed: true},
	}

	if err := WriteOutputs(true, results); err != nil {
		t.Fatalf("WriteOutputs() error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "passed=true") {
		t.Errorf("output should contain 'passed=true', got: %s", content)
	}
	if !strings.Contains(content, "results=") {
		t.Errorf("output should contain 'results=', got: %s", content)
	}
}

func TestWriteJobSummaryNoEnvVar(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	err := WriteJobSummary(nil)
	if err != nil {
		t.Fatalf("should not error when GITHUB_STEP_SUMMARY is empty: %v", err)
	}
}

func TestWriteOutputsNoEnvVar(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")
	err := WriteOutputs(true, nil)
	if err != nil {
		t.Fatalf("should not error when GITHUB_OUTPUT is empty: %v", err)
	}
}
