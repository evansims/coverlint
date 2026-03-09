package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJacoco(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantLine     *Metric
		wantBranch   *Metric
		wantFunction *Metric
		wantErr      bool
	}{
		{
			name:         "basic coverage",
			fixture:      "jacoco/basic.xml",
			wantLine:     &Metric{Hit: 170, Total: 178},
			wantBranch:   &Metric{Hit: 35, Total: 40},
			wantFunction: &Metric{Hit: 32, Total: 35},
		},
		{
			name:         "with sourcefiles and line detail",
			fixture:      "jacoco/with_sourcefiles.xml",
			wantLine:     &Metric{Hit: 4, Total: 6},
			wantBranch:   &Metric{Hit: 3, Total: 5},
			wantFunction: &Metric{Hit: 3, Total: 4},
		},
		{
			name:    "no counters errors",
			fixture: "jacoco/no_counters.xml",
			wantErr: true,
		},
		{
			name:    "no LINE counter errors",
			fixture: "jacoco/no_line_counter.xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", tt.fixture))
			if err != nil {
				t.Fatal(err)
			}
			result, err := parseJacoco(data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseJacoco() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			assertMetric(t, "line", result.Line, tt.wantLine)
			assertMetric(t, "branch", result.Branch, tt.wantBranch)
			assertMetric(t, "function", result.Function, tt.wantFunction)
		})
	}
}

func TestParseJacocoFileDetails(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "jacoco", "with_sourcefiles.xml"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseJacoco(data)
	if err != nil {
		t.Fatal(err)
	}

	// Verify FileDetails are populated from sourcefile line elements
	if len(result.FileDetails) != 2 {
		t.Fatalf("expected 2 file details, got %d", len(result.FileDetails))
	}

	fooDetail := result.FileDetails["com/example/Foo.java"]
	if fooDetail == nil {
		t.Fatal("expected detail for com/example/Foo.java")
	}

	// Line 10: ci=5 (covered), so should be > 0
	if fooDetail.Lines[10] != 5 {
		t.Errorf("line 10 count = %d, want 5", fooDetail.Lines[10])
	}
	// Line 11: ci=0 (uncovered)
	if fooDetail.Lines[11] != 0 {
		t.Errorf("line 11 count = %d, want 0", fooDetail.Lines[11])
	}

	// Line 10 has branches (cb=2, mb=0), should have branch entries
	if len(fooDetail.Branches) == 0 {
		t.Error("expected branch entries for Foo.java")
	}

	// Verify methods tracked from sourcefile-level METHOD counter
	if len(fooDetail.Functions) == 0 {
		t.Error("expected function entries for Foo.java")
	}

	// Verify Files (per-file summaries for suggestions)
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files for suggestions, got %d", len(result.Files))
	}

	// Find Foo.java in files
	var fooFile *FileCoverage
	for i := range result.Files {
		if result.Files[i].Path == "com/example/Foo.java" {
			fooFile = &result.Files[i]
			break
		}
	}
	if fooFile == nil {
		t.Fatal("expected Foo.java in files")
	}
	if fooFile.Line == nil {
		t.Fatal("expected line metric for Foo.java")
	}
	if fooFile.Branch == nil {
		t.Fatal("expected branch metric for Foo.java")
	}
	if fooFile.Function == nil {
		t.Fatal("expected function metric for Foo.java")
	}
}

func TestParseJacocoBarFileNoBranches(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "jacoco", "with_sourcefiles.xml"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseJacoco(data)
	if err != nil {
		t.Fatal(err)
	}

	// Bar.java has no branches in its lines — find it in Files
	var barFile *FileCoverage
	for i := range result.Files {
		if result.Files[i].Path == "com/example/Bar.java" {
			barFile = &result.Files[i]
			break
		}
	}
	if barFile == nil {
		t.Fatal("expected Bar.java in files")
	}
	if barFile.Branch != nil {
		t.Error("expected nil branch metric for Bar.java (no branches)")
	}
}

func TestParseJacocoInvalidXML(t *testing.T) {
	_, err := parseJacoco([]byte("not xml at all"))
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
	if !strings.Contains(err.Error(), "parsing jacoco XML") {
		t.Errorf("error should mention parsing: %v", err)
	}
}

func TestParseJacocoRejectsEntities(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><!DOCTYPE report [<!ENTITY a "x">]><report/>`)
	_, err := parseJacoco(data)
	if err == nil {
		t.Fatal("expected error for XML with ENTITY")
	}
	if !strings.Contains(err.Error(), "ENTITY") {
		t.Errorf("error should mention ENTITY: %v", err)
	}
}

func TestParseJacocoZeroBranch(t *testing.T) {
	// Report with zero branches should not set Branch metric
	data := []byte(`<?xml version="1.0"?>
<report name="NoBranch">
  <counter type="LINE" missed="10" covered="90"/>
  <counter type="BRANCH" missed="0" covered="0"/>
  <counter type="METHOD" missed="2" covered="8"/>
</report>`)
	result, err := parseJacoco(data)
	if err != nil {
		t.Fatal(err)
	}
	if result.Branch != nil {
		t.Error("expected nil Branch when total is 0")
	}
}

func TestParseJacocoZeroMethod(t *testing.T) {
	// Report with zero methods should not set Function metric
	data := []byte(`<?xml version="1.0"?>
<report name="NoMethod">
  <counter type="LINE" missed="10" covered="90"/>
  <counter type="METHOD" missed="0" covered="0"/>
</report>`)
	result, err := parseJacoco(data)
	if err != nil {
		t.Fatal(err)
	}
	if result.Function != nil {
		t.Error("expected nil Function when total is 0")
	}
}
