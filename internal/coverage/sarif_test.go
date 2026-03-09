package coverage

import (
	"fmt"
	"testing"
)

func TestGenerateSARIF_WithLineDetails(t *testing.T) {
	files := []FileCoverage{
		{Path: "src/main.go", Line: &Metric{Hit: 8, Total: 10}},
	}
	fileDetails := map[string]*FileLineDetail{
		"src/main.go": {
			Lines: map[int]int64{
				1:  1,
				2:  1,
				3:  0, // uncovered
				4:  1,
				5:  0, // uncovered
				6:  1,
				7:  1,
				8:  1,
				9:  1,
				10: 1,
			},
		},
	}

	doc := GenerateSARIF(files, fileDetails, nil)

	if doc.Version != "2.1.0" {
		t.Errorf("version = %q, want %q", doc.Version, "2.1.0")
	}
	if doc.Schema != "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json" {
		t.Errorf("unexpected schema: %s", doc.Schema)
	}
	if len(doc.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(doc.Runs))
	}

	run := doc.Runs[0]
	if run.Tool.Driver.Name != "coverlint" {
		t.Errorf("driver name = %q, want %q", run.Tool.Driver.Name, "coverlint")
	}
	if len(run.Tool.Driver.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(run.Tool.Driver.Rules))
	}

	// Should have 2 uncovered lines (lines 3 and 5)
	if len(run.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(run.Results))
	}

	// Results should be sorted by path and line
	r0 := run.Results[0]
	if r0.RuleID != "coverage/uncovered-line" {
		t.Errorf("result[0] ruleId = %q, want %q", r0.RuleID, "coverage/uncovered-line")
	}
	loc0 := r0.Locations[0].PhysicalLocation
	if loc0.ArtifactLocation.URI != "src/main.go" {
		t.Errorf("result[0] uri = %q, want %q", loc0.ArtifactLocation.URI, "src/main.go")
	}
	if loc0.Region == nil || loc0.Region.StartLine != 3 {
		t.Errorf("result[0] startLine = %v, want 3", loc0.Region)
	}

	r1 := run.Results[1]
	loc1 := r1.Locations[0].PhysicalLocation
	if loc1.Region == nil || loc1.Region.StartLine != 5 {
		t.Errorf("result[1] startLine = %v, want 5", loc1.Region)
	}
}

func TestGenerateSARIF_WithBlockDetails(t *testing.T) {
	files := []FileCoverage{
		{Path: "pkg/handler.go", Line: &Metric{Hit: 5, Total: 10}},
	}
	blockDetails := map[string]map[string]*BlockEntry{
		"pkg/handler.go": {
			"pkg/handler.go:5.1,10.1":  {Stmts: 3, Count: 0}, // uncovered
			"pkg/handler.go:15.1,20.1": {Stmts: 2, Count: 5}, // covered
			"pkg/handler.go:25.1,30.1": {Stmts: 4, Count: 0}, // uncovered
		},
	}

	doc := GenerateSARIF(files, nil, blockDetails)

	if len(doc.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(doc.Runs))
	}

	run := doc.Runs[0]
	// Should have 2 uncovered blocks
	if len(run.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(run.Results))
	}

	// Results should be sorted
	r0 := run.Results[0]
	if r0.RuleID != "coverage/uncovered-block" {
		t.Errorf("result[0] ruleId = %q, want %q", r0.RuleID, "coverage/uncovered-block")
	}
	loc0 := r0.Locations[0].PhysicalLocation
	if loc0.Region == nil || loc0.Region.StartLine != 5 || loc0.Region.EndLine != 10 {
		t.Errorf("result[0] region = %v, want startLine=5, endLine=10", loc0.Region)
	}

	r1 := run.Results[1]
	loc1 := r1.Locations[0].PhysicalLocation
	if loc1.Region == nil || loc1.Region.StartLine != 25 || loc1.Region.EndLine != 30 {
		t.Errorf("result[1] region = %v, want startLine=25, endLine=30", loc1.Region)
	}
}

func TestGenerateSARIF_Empty(t *testing.T) {
	doc := GenerateSARIF(nil, nil, nil)

	if len(doc.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(doc.Runs))
	}
	if len(doc.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(doc.Runs[0].Results))
	}
}

func TestGenerateSARIF_ResultsCapped(t *testing.T) {
	// Create file details with more than maxSARIFResults uncovered lines
	lines := make(map[int]int64)
	for i := 1; i <= maxSARIFResults+500; i++ {
		lines[i] = 0 // all uncovered
	}
	fileDetails := map[string]*FileLineDetail{
		"big.go": {Lines: lines},
	}
	files := []FileCoverage{
		{Path: "big.go", Line: &Metric{Hit: 0, Total: int64(len(lines))}},
	}

	doc := GenerateSARIF(files, fileDetails, nil)

	if len(doc.Runs[0].Results) != maxSARIFResults {
		t.Errorf("results = %d, want %d (capped)", len(doc.Runs[0].Results), maxSARIFResults)
	}
}

func TestGenerateSARIF_MultipleFiles(t *testing.T) {
	files := []FileCoverage{
		{Path: "b.go", Line: &Metric{Hit: 1, Total: 2}},
		{Path: "a.go", Line: &Metric{Hit: 0, Total: 1}},
	}
	fileDetails := map[string]*FileLineDetail{
		"a.go": {Lines: map[int]int64{1: 0}},
		"b.go": {Lines: map[int]int64{1: 1, 2: 0}},
	}

	doc := GenerateSARIF(files, fileDetails, nil)

	// Should have 2 results (one from a.go line 1, one from b.go line 2)
	if len(doc.Runs[0].Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(doc.Runs[0].Results))
	}

	// Should be sorted: a.go before b.go
	if doc.Runs[0].Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI != "a.go" {
		t.Error("results should be sorted by path, expected a.go first")
	}
}

func TestParseBlockRange(t *testing.T) {
	tests := []struct {
		key       string
		wantStart int
		wantEnd   int
		wantErr   bool
	}{
		{"file.go:5.1,10.1", 5, 10, false},
		{"pkg/handler.go:15.3,20.8", 15, 20, false},
		{"a.go:1.1,1.1", 1, 1, false},
		{"invalid", 0, 0, true},
		{"file.go:bad", 0, 0, true},
		{"file.go:5.1", 0, 0, true},            // missing comma
		{"file.go:abc.1,10.1", 0, 0, true},     // non-numeric start
		{"file.go:5.1,abc.1", 0, 0, true},      // non-numeric end
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			start, end, err := parseBlockRange(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBlockRange(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if start != tt.wantStart || end != tt.wantEnd {
					t.Errorf("parseBlockRange(%q) = (%d, %d), want (%d, %d)", tt.key, start, end, tt.wantStart, tt.wantEnd)
				}
			}
		})
	}
}

func TestGenerateSARIF_ResultMessage(t *testing.T) {
	files := []FileCoverage{
		{Path: "main.go", Line: &Metric{Hit: 0, Total: 1}},
	}
	fileDetails := map[string]*FileLineDetail{
		"main.go": {Lines: map[int]int64{10: 0}},
	}

	doc := GenerateSARIF(files, fileDetails, nil)

	if len(doc.Runs[0].Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(doc.Runs[0].Results))
	}

	msg := doc.Runs[0].Results[0].Message.Text
	expected := fmt.Sprintf("Line %d is not covered by tests", 10)
	if msg != expected {
		t.Errorf("message = %q, want %q", msg, expected)
	}
}
