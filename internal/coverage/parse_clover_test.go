package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseClover(t *testing.T) {
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
			fixture:      "clover/basic.xml",
			wantLine:     &Metric{Hit: 170, Total: 200},
			wantBranch:   &Metric{Hit: 35, Total: 50},
			wantFunction: &Metric{Hit: 32, Total: 40},
		},
		{
			name:         "with line elements in packages",
			fixture:      "clover/with_lines.xml",
			wantLine:     &Metric{Hit: 7, Total: 10},
			wantBranch:   &Metric{Hit: 2, Total: 4},
			wantFunction: &Metric{Hit: 2, Total: 3},
		},
		{
			name:         "top-level files without packages",
			fixture:      "clover/top_level_files.xml",
			wantLine:     &Metric{Hit: 3, Total: 5},
			wantBranch:   nil,
			wantFunction: &Metric{Hit: 1, Total: 1},
		},
		{
			name:    "no statements errors",
			fixture: "clover/no_statements.xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", tt.fixture))
			if err != nil {
				t.Fatal(err)
			}
			result, err := parseClover(data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseClover() error = %v, wantErr %v", err, tt.wantErr)
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

func TestParseCloverFileDetails(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "clover", "with_lines.xml"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseClover(data)
	if err != nil {
		t.Fatal(err)
	}

	// Verify FileDetails are populated from line elements
	if len(result.FileDetails) != 2 {
		t.Fatalf("expected 2 file details, got %d", len(result.FileDetails))
	}

	fooDetail := result.FileDetails["src/Foo.php"]
	if fooDetail == nil {
		t.Fatal("expected detail for src/Foo.php")
	}

	// stmt lines
	if fooDetail.Lines[11] != 3 {
		t.Errorf("line 11 count = %d, want 3", fooDetail.Lines[11])
	}
	if fooDetail.Lines[12] != 0 {
		t.Errorf("line 12 count = %d, want 0", fooDetail.Lines[12])
	}

	// cond lines should appear in both Lines and Branches
	if fooDetail.Lines[13] != 2 {
		t.Errorf("cond line 13 count in Lines = %d, want 2", fooDetail.Lines[13])
	}
	if fooDetail.Branches["13"] != 2 {
		t.Errorf("cond line 13 count in Branches = %d, want 2", fooDetail.Branches["13"])
	}

	// method lines should appear in Lines and Functions
	if fooDetail.Lines[10] != 5 {
		t.Errorf("method line 10 count in Lines = %d, want 5", fooDetail.Lines[10])
	}
	funcKey := "src/Foo.php:10"
	if fooDetail.Functions[funcKey] != 5 {
		t.Errorf("function key %q count = %d, want 5", funcKey, fooDetail.Functions[funcKey])
	}

	// Verify Files (per-file summaries for suggestions)
	if len(result.Files) != 2 {
		t.Fatalf("expected 2 files for suggestions, got %d", len(result.Files))
	}
}

func TestParseCloverTopLevelFileDetails(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "clover", "top_level_files.xml"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseClover(data)
	if err != nil {
		t.Fatal(err)
	}

	// Top-level files use Name when Path is empty
	detail := result.FileDetails["main.py"]
	if detail == nil {
		t.Fatal("expected detail for main.py")
	}
	// Only lines with count > 0 get tracked in the Lines map (max semantics):
	// line 1 (count=1), line 2 (count=1), line 5 (method count=1) = 3 entries
	if len(detail.Lines) != 3 {
		t.Errorf("expected 3 lines with count > 0, got %d", len(detail.Lines))
	}
}

func TestParseCloverInvalidXML(t *testing.T) {
	_, err := parseClover([]byte("not xml at all"))
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
	if !strings.Contains(err.Error(), "parsing clover XML") {
		t.Errorf("error should mention parsing: %v", err)
	}
}

func TestParseCloverRejectsEntities(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><!DOCTYPE coverage [<!ENTITY a "x">]><coverage/>`)
	_, err := parseClover(data)
	if err == nil {
		t.Fatal("expected error for XML with ENTITY")
	}
	if !strings.Contains(err.Error(), "ENTITY") {
		t.Errorf("error should mention ENTITY: %v", err)
	}
}
