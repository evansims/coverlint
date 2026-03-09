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

func TestFormatViolationDelta(t *testing.T) {
	v := Violation{
		Entry:    "coverage",
		Metric:   "delta",
		Actual:   -5.0,
		Required: -2.0,
	}
	msg := FormatViolation(v)
	if !strings.Contains(msg, "score changed by -5.0 points") {
		t.Errorf("message should contain delta change, got: %s", msg)
	}
	if !strings.Contains(msg, "minimum allowed change is -2.0") {
		t.Errorf("message should contain min allowed change, got: %s", msg)
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

	if err := WriteJobSummary(results, false, nil); err != nil {
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

	if err := WriteOutputs(true, results, nil, ""); err != nil {
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
	if !strings.Contains(content, "results<<COVERLINT_RESULTS_EOF") {
		t.Errorf("output should contain multiline results delimiter, got: %s", content)
	}
	if !strings.Contains(content, `"backend"`) {
		t.Errorf("output should contain results JSON, got: %s", content)
	}
}

func TestWriteOutputsWithBadge(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)

	score := 85.0
	line := 90.0
	results := []EntryResult{
		{Name: "total", Score: &score, Line: &line, Passed: true},
	}

	if err := WriteOutputs(true, results, nil, ""); err != nil {
		t.Fatalf("WriteOutputs() error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Should have badge SVG output
	if !strings.Contains(content, "badge-svg<<COVERLINT_SVG_EOF") {
		t.Error("output should contain badge-svg with SVG delimiter")
	}
	if !strings.Contains(content, "<svg") {
		t.Error("output should contain SVG content")
	}

	// Should have badge JSON output
	if !strings.Contains(content, "badge-json=") {
		t.Error("output should contain badge-json")
	}
	if !strings.Contains(content, `"coverage"`) {
		t.Error("badge-json should contain coverage label")
	}
	// Badge should use rounded whole numbers
	if !strings.Contains(content, `"85%"`) {
		t.Error("badge-json should contain rounded percentage '85%'")
	}
}

func TestWriteOutputsPassedFalse(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []EntryResult{
		{Name: "backend", Passed: false},
	}

	if err := WriteOutputs(false, results, nil, ""); err != nil {
		t.Fatalf("WriteOutputs() error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "passed=false") {
		t.Errorf("output should contain 'passed=false', got: %s", content)
	}
}

func TestWriteOutputsNoScore(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)

	// Entry without Score — should not produce badge outputs
	results := []EntryResult{
		{Name: "test", Passed: true},
	}

	if err := WriteOutputs(true, results, nil, ""); err != nil {
		t.Fatalf("WriteOutputs() error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if strings.Contains(content, "badge-svg") {
		t.Error("should not contain badge-svg when no score")
	}
	if strings.Contains(content, "badge-json") {
		t.Error("should not contain badge-json when no score")
	}
}

func TestWriteOutputsEmptyResults(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)

	if err := WriteOutputs(true, nil, nil, ""); err != nil {
		t.Fatalf("WriteOutputs() error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "passed=true") {
		t.Error("should still write passed output")
	}
}

func TestWriteJobSummaryOmitsUnsupportedColumns(t *testing.T) {
	summaryFile := filepath.Join(t.TempDir(), "summary.md")
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	line := 100.0
	results := []EntryResult{
		{
			Name:   "go-coverage",
			Line:   &line,
			Passed: true,
		},
	}

	if err := WriteJobSummary(results, false, nil); err != nil {
		t.Fatalf("WriteJobSummary() error: %v", err)
	}

	data, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if strings.Contains(content, "Branch") {
		t.Error("summary should not contain Branch column when no results have branch data")
	}
	if strings.Contains(content, "Function") {
		t.Error("summary should not contain Function column when no results have function data")
	}
	if !strings.Contains(content, "Line") {
		t.Error("summary should contain Line column")
	}
	if !strings.Contains(content, "100.0%") {
		t.Error("summary should contain line percentage")
	}
}

func TestWriteJobSummaryMultiFormatTotal(t *testing.T) {
	summaryFile := filepath.Join(t.TempDir(), "summary.md")
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	goLine := 90.0
	nodeLine := 85.0
	nodeBranch := 70.0
	totalLine := 87.5
	totalBranch := 70.0

	results := []EntryResult{
		{Name: "gocover", Line: &goLine, Passed: true},
		{Name: "lcov", Line: &nodeLine, Branch: &nodeBranch, Passed: true},
		{Name: "Total", Line: &totalLine, Branch: &totalBranch, Passed: true},
	}

	if err := WriteJobSummary(results, true, nil); err != nil {
		t.Fatalf("WriteJobSummary() error: %v", err)
	}

	data, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Should have per-format rows
	if !strings.Contains(content, "| gocover") {
		t.Error("summary should contain gocover row")
	}
	if !strings.Contains(content, "| lcov") {
		t.Error("summary should contain lcov row")
	}

	// Total row should be bold
	if !strings.Contains(content, "**Total**") {
		t.Error("summary should contain bold Total row")
	}
	if !strings.Contains(content, "**87.5%**") {
		t.Error("summary should contain bold total percentage")
	}
}

func TestWriteJobSummaryWithSuggestions(t *testing.T) {
	summaryFile := filepath.Join(t.TempDir(), "summary.md")
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	line := 60.0
	results := []EntryResult{
		{Name: "test", Line: &line, Passed: true},
	}
	suggestions := []Suggestion{
		{Path: "big.go", UncoveredLines: 50, TotalLines: 100, LinePct: 50.0},
	}

	if err := WriteJobSummary(results, false, suggestions); err != nil {
		t.Fatalf("WriteJobSummary() error: %v", err)
	}

	data, _ := os.ReadFile(summaryFile)
	content := string(data)
	if !strings.Contains(content, "Top Opportunities") {
		t.Error("summary should contain suggestions section")
	}
	if !strings.Contains(content, "big.go") {
		t.Error("summary should contain suggestion file")
	}
}

func TestWriteJobSummaryTotalPassedFalse(t *testing.T) {
	summaryFile := filepath.Join(t.TempDir(), "summary.md")
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	line1 := 90.0
	totalLine := 85.0

	results := []EntryResult{
		{Name: "gocover", Line: &line1, Passed: true},
		{Name: "Total", Line: &totalLine, Passed: false},
	}

	if err := WriteJobSummary(results, true, nil); err != nil {
		t.Fatalf("WriteJobSummary() error: %v", err)
	}

	data, _ := os.ReadFile(summaryFile)
	content := string(data)

	// Total row should show **Fail** when Passed is false
	if !strings.Contains(content, "**Fail**") {
		t.Error("total row should show Fail when not passed")
	}
}

func TestSanitizeWorkflowCommand(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal message", "normal message"},
		{"has\nnewline", "has newline"},
		{"has\r\nnewline", "has  newline"},
		{"has::colons", "has: :colons"},
		{"inject\n::error::pwned", "inject : :error: :pwned"},
	}
	for _, tt := range tests {
		got := sanitizeWorkflowCommand(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeWorkflowCommand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeMarkdown(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal", "normal"},
		{"has|pipe", "has\\|pipe"},
		{"has\nnewline", "has newline"},
	}
	for _, tt := range tests {
		got := sanitizeMarkdown(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWriteJobSummaryNoEnvVar(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	err := WriteJobSummary(nil, false, nil)
	if err != nil {
		t.Fatalf("should not error when GITHUB_STEP_SUMMARY is empty: %v", err)
	}
}

func TestWriteOutputsNoEnvVar(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "")
	err := WriteOutputs(true, nil, nil, "")
	if err != nil {
		t.Fatalf("should not error when GITHUB_OUTPUT is empty: %v", err)
	}
}

func TestFmtPct(t *testing.T) {
	tests := []struct {
		input *float64
		want  string
	}{
		{nil, "N/A"},
		{floatPtr(0.0), "0.0%"},
		{floatPtr(82.5), "82.5%"},
		{floatPtr(100.0), "100.0%"},
	}
	for _, tt := range tests {
		got := fmtPct(tt.input)
		if got != tt.want {
			t.Errorf("fmtPct(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWriteOutputsWithBaseline(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)

	score := 85.0
	line := 90.0
	results := []EntryResult{
		{Name: "total", Score: &score, Line: &line, Passed: true},
	}

	bl := 90.0
	baseline := &BaselineData{
		Score:     85.0,
		Line:      &bl,
		Timestamp: "2025-01-01T00:00:00Z",
	}

	if err := WriteOutputs(true, results, baseline, ""); err != nil {
		t.Fatalf("WriteOutputs() error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "baseline<<COVERLINT_BASELINE_EOF") {
		t.Error("output should contain baseline with delimiter")
	}
	if !strings.Contains(content, `"score":85`) {
		t.Error("output should contain baseline score")
	}
	if !strings.Contains(content, `"timestamp":"2025-01-01T00:00:00Z"`) {
		t.Error("output should contain baseline timestamp")
	}
}

func TestWriteOutputsWithSARIF(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []EntryResult{
		{Name: "test", Passed: true},
	}

	if err := WriteOutputs(true, results, nil, "/tmp/coverage.sarif"); err != nil {
		t.Fatalf("WriteOutputs() error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if !strings.Contains(content, "sarif=/tmp/coverage.sarif") {
		t.Errorf("output should contain sarif path, got: %s", content)
	}
}

func TestWriteOutputsWithoutSARIF(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)

	results := []EntryResult{
		{Name: "test", Passed: true},
	}

	if err := WriteOutputs(true, results, nil, ""); err != nil {
		t.Fatalf("WriteOutputs() error: %v", err)
	}

	data, _ := os.ReadFile(outputFile)
	content := string(data)

	if strings.Contains(content, "sarif=") {
		t.Errorf("output should not contain sarif when path is empty, got: %s", content)
	}
}

func TestWriteOutputsInvalidPath(t *testing.T) {
	t.Setenv("GITHUB_OUTPUT", "/nonexistent/dir/output")
	err := WriteOutputs(true, nil, nil, "")
	if err == nil {
		t.Fatal("expected error for invalid output path")
	}
	if !strings.Contains(err.Error(), "opening output file") {
		t.Errorf("error should mention opening: %v", err)
	}
}

func TestWriteJobSummaryInvalidPath(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "/nonexistent/dir/summary")
	err := WriteJobSummary(nil, false, nil)
	if err == nil {
		t.Fatal("expected error for invalid summary path")
	}
	if !strings.Contains(err.Error(), "opening step summary file") {
		t.Errorf("error should mention opening: %v", err)
	}
}
