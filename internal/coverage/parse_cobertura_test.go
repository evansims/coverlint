package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCobertura(t *testing.T) {
	tests := []struct {
		name         string
		fixture      string
		wantLine     *Metric
		wantBranch   *Metric
		wantFunction *Metric
		wantErr      bool
	}{
		{
			name:         "basic coverage with methods",
			fixture:      "cobertura/basic.xml",
			wantLine:     &Metric{Hit: 170, Total: 200},
			wantBranch:   &Metric{Hit: 35, Total: 50},
			wantFunction: &Metric{Hit: 2, Total: 3},
		},
		{
			name:       "no branches or methods",
			fixture:    "cobertura/no_branches.xml",
			wantLine:   &Metric{Hit: 90, Total: 100},
			wantBranch: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", tt.fixture))
			if err != nil {
				t.Fatal(err)
			}
			result, err := parseCobertura(data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseCobertura() error = %v, wantErr %v", err, tt.wantErr)
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

func TestParseCoberturaFileDetails(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "cobertura", "basic.xml"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseCobertura(data)
	if err != nil {
		t.Fatal(err)
	}

	if result.FileDetails == nil {
		t.Fatal("expected FileDetails to be populated")
	}
	if len(result.FileDetails) == 0 {
		t.Fatal("expected at least one file in FileDetails")
	}

	// Verify Files for suggestions
	if len(result.Files) == 0 {
		t.Fatal("expected at least one FileCoverage")
	}
}

func TestParseCoberturaNoLines(t *testing.T) {
	data := []byte(`<?xml version="1.0"?>
<coverage lines-valid="0" lines-covered="0" branches-valid="0" branches-covered="0">
  <packages/>
</coverage>`)
	_, err := parseCobertura(data)
	if err == nil {
		t.Fatal("expected error for no line data")
	}
	if !strings.Contains(err.Error(), "no line data") {
		t.Errorf("error should mention no line data: %v", err)
	}
}

func TestParseCoberturaInvalidXML(t *testing.T) {
	_, err := parseCobertura([]byte("not xml"))
	if err == nil {
		t.Fatal("expected error for invalid XML")
	}
	if !strings.Contains(err.Error(), "parsing cobertura XML") {
		t.Errorf("error should mention parsing: %v", err)
	}
}

func TestParseCoberturaRejectsEntities(t *testing.T) {
	data := []byte(`<?xml version="1.0"?><!DOCTYPE coverage [<!ENTITY a "x">]><coverage/>`)
	_, err := parseCobertura(data)
	if err == nil {
		t.Fatal("expected error for XML with ENTITY")
	}
}

func TestParseCoberturaMethodTracking(t *testing.T) {
	data := []byte(`<?xml version="1.0"?>
<coverage lines-valid="10" lines-covered="8" branches-valid="0" branches-covered="0">
  <packages>
    <package>
      <classes>
        <class filename="src/math.py">
          <methods>
            <method name="add">
              <lines>
                <line number="1" hits="5"/>
                <line number="2" hits="3"/>
              </lines>
            </method>
            <method name="subtract">
              <lines>
                <line number="5" hits="0"/>
                <line number="6" hits="0"/>
              </lines>
            </method>
          </methods>
          <lines>
            <line number="1" hits="5"/>
            <line number="2" hits="3"/>
            <line number="5" hits="0"/>
            <line number="6" hits="0"/>
          </lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>`)

	result, err := parseCobertura(data)
	if err != nil {
		t.Fatal(err)
	}

	// add: covered (has hits > 0), subtract: uncovered
	assertMetric(t, "function", result.Function, &Metric{Hit: 1, Total: 2})

	// Verify file details have function tracking
	detail := result.FileDetails["src/math.py"]
	if detail == nil {
		t.Fatal("expected detail for src/math.py")
	}
	if len(detail.Functions) != 2 {
		t.Errorf("expected 2 functions, got %d", len(detail.Functions))
	}
}

func TestParseCoberturaMultipleClassesSameFile(t *testing.T) {
	data := []byte(`<?xml version="1.0"?>
<coverage lines-valid="10" lines-covered="7" branches-valid="0" branches-covered="0">
  <packages>
    <package>
      <classes>
        <class filename="src/utils.py">
          <methods/>
          <lines>
            <line number="1" hits="3"/>
            <line number="2" hits="0"/>
          </lines>
        </class>
        <class filename="src/utils.py">
          <methods/>
          <lines>
            <line number="2" hits="5"/>
            <line number="3" hits="1"/>
          </lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>`)

	result, err := parseCobertura(data)
	if err != nil {
		t.Fatal(err)
	}

	detail := result.FileDetails["src/utils.py"]
	if detail == nil {
		t.Fatal("expected detail for src/utils.py")
	}

	// Line 2 should take max(0, 5) = 5
	if detail.Lines[2] != 5 {
		t.Errorf("line 2 count = %d, want 5 (max of both classes)", detail.Lines[2])
	}
}
