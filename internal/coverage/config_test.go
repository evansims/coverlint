package coverage

import (
	"strings"
	"testing"
)

func TestParseInputs(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantErr     string
		wantFormats []string
	}{
		{
			name: "valid minimal",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "gocover",
				"INPUT_MIN-LINE": "80",
			},
			wantFormats: []string{"gocover"},
		},
		{
			name: "valid all thresholds",
			env: map[string]string{
				"INPUT_PATH":               "lcov.info",
				"INPUT_FORMAT":             "lcov",
					"INPUT_MIN-LINE":     "80",
				"INPUT_MIN-BRANCH":   "70",
				"INPUT_MIN-FUNCTION": "75",
			},
			wantFormats: []string{"lcov"},
		},
		{
			name: "path optional",
			env: map[string]string{
				"INPUT_FORMAT":         "lcov",
				"INPUT_MIN-LINE": "80",
			},
			wantFormats: []string{"lcov"},
		},
		{
			name: "multiple formats",
			env: map[string]string{
				"INPUT_FORMAT":         "gocover,lcov",
				"INPUT_MIN-LINE": "80",
			},
			wantFormats: []string{"gocover", "lcov"},
		},
		{
			name: "multiple formats with spaces",
			env: map[string]string{
				"INPUT_FORMAT":         "gocover, lcov, cobertura",
				"INPUT_MIN-LINE": "80",
			},
			wantFormats: []string{"gocover", "lcov", "cobertura"},
		},
		{
			name: "multiple formats newline-separated",
			env: map[string]string{
				"INPUT_FORMAT":         "gocover\nlcov\ncobertura",
				"INPUT_MIN-LINE": "80",
			},
			wantFormats: []string{"gocover", "lcov", "cobertura"},
		},
		{
			name: "mixed newlines and commas",
			env: map[string]string{
				"INPUT_FORMAT":         "gocover,lcov\ncobertura",
				"INPUT_MIN-LINE": "80",
			},
			wantFormats: []string{"gocover", "lcov", "cobertura"},
		},
		{
			name: "format auto-detected when omitted",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_MIN-LINE": "80",
			},
			wantFormats: formatOrder,
		},
		{
			name: "invalid format",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "invalid",
				"INPUT_MIN-LINE": "80",
			},
			wantErr: "not valid",
		},
		{
			name: "one invalid in multi-format",
			env: map[string]string{
				"INPUT_FORMAT":         "gocover,invalid",
				"INPUT_MIN-LINE": "80",
			},
			wantErr: "not valid",
		},
		{
			name: "no thresholds is valid",
			env: map[string]string{
				"INPUT_PATH":   "cover.out",
				"INPUT_FORMAT": "gocover",
			},
			wantFormats: []string{"gocover"},
		},
		{
			name: "negative threshold",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "lcov",
				"INPUT_MIN-LINE": "-5",
			},
			wantErr: "between 0 and 100",
		},
		{
			name: "threshold over 100",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "lcov",
				"INPUT_MIN-LINE": "200",
			},
			wantErr: "between 0 and 100",
		},
		{
			name: "non-numeric threshold",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "lcov",
				"INPUT_MIN-LINE": "abc",
			},
			wantErr: "not a valid number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all input env vars
			for _, key := range []string{
				"INPUT_PATH", "INPUT_FORMAT",
				"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
				"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
				"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
				"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
			} {
				t.Setenv(key, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			inp, err := ParseInputs()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if inp.Path != tt.env["INPUT_PATH"] {
				t.Errorf("path = %q, want %q", inp.Path, tt.env["INPUT_PATH"])
			}
			if len(inp.Formats) != len(tt.wantFormats) {
				t.Fatalf("formats = %v, want %v", inp.Formats, tt.wantFormats)
			}
			for i, f := range inp.Formats {
				if f != tt.wantFormats[i] {
					t.Errorf("formats[%d] = %q, want %q", i, f, tt.wantFormats[i])
				}
			}
		})
	}
}

func TestParseInputsAutoFormat(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}

	t.Setenv("INPUT_PATH", "cover.out")
	t.Setenv("INPUT_MIN-LINE", "80")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !inp.AutoFormat {
		t.Error("expected AutoFormat to be true when format is omitted")
	}
	if len(inp.Formats) != len(formatOrder) {
		t.Errorf("expected %d formats, got %d", len(formatOrder), len(inp.Formats))
	}

	// Verify explicit format sets AutoFormat to false
	t.Setenv("INPUT_FORMAT", "lcov")
	inp, err = ParseInputs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inp.AutoFormat {
		t.Error("expected AutoFormat to be false when format is specified")
	}
}

func TestParseInputsDefaults(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}

	t.Setenv("INPUT_PATH", "cover.out")
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_MIN-LINE", "80")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inp.WorkDir != "." {
		t.Errorf("workdir should default to '.', got %q", inp.WorkDir)
	}
	if !inp.FailOnError {
		t.Error("fail-on-error should default to true")
	}
}

func TestParseInputsMinCoverage(t *testing.T) {
	clear := func(t *testing.T) {
		t.Helper()
		for _, key := range []string{
			"INPUT_PATH", "INPUT_FORMAT",
			"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
			"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
			"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
			"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
			"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
		} {
			t.Setenv(key, "")
		}
	}

	t.Run("sets weighted score threshold only", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_MIN-COVERAGE", "80")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// min-coverage sets the weighted score threshold, not individual metrics
		if inp.Threshold.MinCoverage == nil || *inp.Threshold.MinCoverage != 80 {
			t.Errorf("MinCoverage = %v, want 80", inp.Threshold.MinCoverage)
		}
		// Individual metrics should remain nil
		if inp.Threshold.Line != nil {
			t.Errorf("Line = %v, want nil", inp.Threshold.Line)
		}
		if inp.Threshold.Branch != nil {
			t.Errorf("Branch = %v, want nil", inp.Threshold.Branch)
		}
		if inp.Threshold.Function != nil {
			t.Errorf("Function = %v, want nil", inp.Threshold.Function)
		}
	})

	t.Run("min-coverage with individual hard floors", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_MIN-COVERAGE", "80")
		t.Setenv("INPUT_MIN-BRANCH", "60")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.Threshold.MinCoverage == nil || *inp.Threshold.MinCoverage != 80 {
			t.Errorf("MinCoverage = %v, want 80", inp.Threshold.MinCoverage)
		}
		if inp.Threshold.Branch == nil || *inp.Threshold.Branch != 60 {
			t.Errorf("Branch = %v, want 60", inp.Threshold.Branch)
		}
		// Line and Function should remain nil (no individual floor set)
		if inp.Threshold.Line != nil {
			t.Errorf("Line = %v, want nil", inp.Threshold.Line)
		}
		if inp.Threshold.Function != nil {
			t.Errorf("Function = %v, want nil", inp.Threshold.Function)
		}
	})

	t.Run("individual thresholds without min-coverage", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_MIN-LINE", "90")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.Threshold.MinCoverage != nil {
			t.Errorf("MinCoverage = %v, want nil", inp.Threshold.MinCoverage)
		}
		if inp.Threshold.Line == nil || *inp.Threshold.Line != 90 {
			t.Errorf("Line = %v, want 90", inp.Threshold.Line)
		}
		if inp.Threshold.Branch != nil {
			t.Errorf("Branch = %v, want nil", inp.Threshold.Branch)
		}
		if inp.Threshold.Function != nil {
			t.Errorf("Function = %v, want nil", inp.Threshold.Function)
		}
	})

	t.Run("invalid min-coverage", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_MIN-COVERAGE", "abc")

		_, err := ParseInputs()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "min-coverage") {
			t.Errorf("error %q should mention min-coverage", err.Error())
		}
	})
}

func TestParseInputsWeights(t *testing.T) {
	clear := func(t *testing.T) {
		t.Helper()
		for _, key := range []string{
			"INPUT_PATH", "INPUT_FORMAT",
			"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
			"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
			"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
			"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
			"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
		} {
			t.Setenv(key, "")
		}
	}

	t.Run("defaults", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		dw := DefaultWeights()
		if inp.Threshold.Weights.Line != dw.Line {
			t.Errorf("weight-line = %v, want %v", inp.Threshold.Weights.Line, dw.Line)
		}
		if inp.Threshold.Weights.Branch != dw.Branch {
			t.Errorf("weight-branch = %v, want %v", inp.Threshold.Weights.Branch, dw.Branch)
		}
		if inp.Threshold.Weights.Function != dw.Function {
			t.Errorf("weight-function = %v, want %v", inp.Threshold.Weights.Function, dw.Function)
		}
	})

	t.Run("custom weights", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_WEIGHT-LINE", "100")
		t.Setenv("INPUT_WEIGHT-BRANCH", "0")
		t.Setenv("INPUT_WEIGHT-FUNCTION", "0")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.Threshold.Weights.Line != 100 {
			t.Errorf("weight-line = %v, want 100", inp.Threshold.Weights.Line)
		}
		if inp.Threshold.Weights.Branch != 0 {
			t.Errorf("weight-branch = %v, want 0", inp.Threshold.Weights.Branch)
		}
		if inp.Threshold.Weights.Function != 0 {
			t.Errorf("weight-function = %v, want 0", inp.Threshold.Weights.Function)
		}
	})

	t.Run("invalid weight", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_WEIGHT-LINE", "abc")

		_, err := ParseInputs()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "weight-line") {
			t.Errorf("error %q should mention weight-line", err.Error())
		}
	})

	t.Run("invalid weight-branch", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_WEIGHT-BRANCH", "xyz")

		_, err := ParseInputs()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "weight-branch") {
			t.Errorf("error %q should mention weight-branch", err.Error())
		}
	})

	t.Run("invalid weight-function", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_WEIGHT-FUNCTION", "xyz")

		_, err := ParseInputs()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "weight-function") {
			t.Errorf("error %q should mention weight-function", err.Error())
		}
	})
}

func TestParseInputsInvalidMinBranch(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_MIN-BRANCH", "abc")

	_, err := ParseInputs()
	if err == nil {
		t.Fatal("expected error for invalid min-branch")
	}
	if !strings.Contains(err.Error(), "min-branch") {
		t.Errorf("error should mention min-branch: %v", err)
	}
}

func TestParseInputsInvalidMinFunction(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_MIN-FUNCTION", "abc")

	_, err := ParseInputs()
	if err == nil {
		t.Fatal("expected error for invalid min-function")
	}
	if !strings.Contains(err.Error(), "min-function") {
		t.Errorf("error should mention min-function: %v", err)
	}
}

func TestParseInputsSuggestionsDefault(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatal(err)
	}
	if !inp.Suggestions {
		t.Error("suggestions should default to true")
	}
}

func TestParseInputsSuggestionsDisabled(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_SUGGESTIONS", "false")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatal(err)
	}
	if inp.Suggestions {
		t.Error("suggestions should be false when set to 'false'")
	}
}

func TestParseInputsAnnotationsDefault(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatal(err)
	}
	if inp.Annotations.Mode != "all" {
		t.Errorf("annotations mode = %q, want %q", inp.Annotations.Mode, "all")
	}
}

func TestParseInputsAnnotationsFalse(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_ANNOTATIONS", "false")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatal(err)
	}
	if inp.Annotations.Mode != "none" {
		t.Errorf("annotations mode = %q, want %q", inp.Annotations.Mode, "none")
	}
}

func TestParseInputsAnnotationsIntegerCap(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_ANNOTATIONS", "10")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatal(err)
	}
	if inp.Annotations.Mode != "limited" {
		t.Errorf("annotations mode = %q, want %q", inp.Annotations.Mode, "limited")
	}
	if inp.Annotations.MaxCount != 10 {
		t.Errorf("annotations max count = %d, want 10", inp.Annotations.MaxCount)
	}
}

func TestParseInputsAnnotationsInvalid(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_ANNOTATIONS", "abc")

	_, err := ParseInputs()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "annotations") {
		t.Errorf("error %q should mention annotations", err.Error())
	}
}

func TestParseInputsAnnotationsNegative(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
		"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_ANNOTATIONS", "-1")

	_, err := ParseInputs()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "annotations") {
		t.Errorf("error %q should mention annotations", err.Error())
	}
}

func TestSplitList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{"a\nb\nc", []string{"a", "b", "c"}},
		{"a,b\nc", []string{"a", "b", "c"}},
		{" a , b ", []string{"a", "b"}},
		{"a,,b", []string{"a", "b"}},
		{"\n\n", nil},
		{"single", []string{"single"}},
	}
	for _, tt := range tests {
		got := splitList(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitList(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitList(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestGetInput(t *testing.T) {
	t.Run("returns env value", func(t *testing.T) {
		t.Setenv("INPUT_TEST-KEY", "myvalue")
		got := getInput("TEST-KEY", "default")
		if got != "myvalue" {
			t.Errorf("got %q, want %q", got, "myvalue")
		}
	})

	t.Run("returns default when empty", func(t *testing.T) {
		t.Setenv("INPUT_TEST-KEY2", "")
		got := getInput("TEST-KEY2", "default")
		if got != "default" {
			t.Errorf("got %q, want %q", got, "default")
		}
	})
}

func TestParseOptionalFloat(t *testing.T) {
	tests := []struct {
		input   string
		want    *float64
		wantErr bool
	}{
		{"", nil, false},
		{"  ", nil, false},
		{"50", floatPtr(50), false},
		{"0", floatPtr(0), false},
		{"100", floatPtr(100), false},
		{"75.5", floatPtr(75.5), false},
		{"-1", nil, true},
		{"101", nil, true},
		{"abc", nil, true},
	}
	for _, tt := range tests {
		got, err := parseOptionalFloat(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseOptionalFloat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if tt.want == nil && got != nil {
			t.Errorf("parseOptionalFloat(%q) = %v, want nil", tt.input, got)
		} else if tt.want != nil && (got == nil || *got != *tt.want) {
			t.Errorf("parseOptionalFloat(%q) = %v, want %v", tt.input, got, *tt.want)
		}
	}
}

func TestParseInputsBaseline(t *testing.T) {
	clear := func(t *testing.T) {
		t.Helper()
		for _, key := range []string{
			"INPUT_PATH", "INPUT_FORMAT",
			"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
			"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
			"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
			"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
			"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
		} {
			t.Setenv(key, "")
		}
	}

	t.Run("baseline path", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_BASELINE", "/path/to/baseline.json")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.Baseline != "/path/to/baseline.json" {
			t.Errorf("Baseline = %q, want /path/to/baseline.json", inp.Baseline)
		}
	})

	t.Run("min-delta positive", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_MIN-DELTA", "1.5")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.MinDelta == nil || *inp.MinDelta != 1.5 {
			t.Errorf("MinDelta = %v, want 1.5", inp.MinDelta)
		}
	})

	t.Run("min-delta negative", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_MIN-DELTA", "-3.0")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.MinDelta == nil || *inp.MinDelta != -3.0 {
			t.Errorf("MinDelta = %v, want -3.0", inp.MinDelta)
		}
	})

	t.Run("min-delta zero", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_MIN-DELTA", "0")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.MinDelta == nil || *inp.MinDelta != 0 {
			t.Errorf("MinDelta = %v, want 0", inp.MinDelta)
		}
	})

	t.Run("min-delta invalid", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_MIN-DELTA", "abc")

		_, err := ParseInputs()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "min-delta") {
			t.Errorf("error %q should mention min-delta", err.Error())
		}
	})

	t.Run("min-delta empty is nil", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.MinDelta != nil {
			t.Errorf("MinDelta = %v, want nil", inp.MinDelta)
		}
	})

	t.Run("baseline empty is empty string", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.Baseline != "" {
			t.Errorf("Baseline = %q, want empty", inp.Baseline)
		}
	})
}

func TestParseInputsSARIF(t *testing.T) {
	clear := func(t *testing.T) {
		t.Helper()
		for _, key := range []string{
			"INPUT_PATH", "INPUT_FORMAT",
			"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
			"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
			"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
			"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
			"INPUT_BASELINE", "INPUT_MIN-DELTA", "INPUT_SARIF",
		} {
			t.Setenv(key, "")
		}
	}

	t.Run("sarif path set", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")
		t.Setenv("INPUT_SARIF", "coverage.sarif")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.SARIF != "coverage.sarif" {
			t.Errorf("SARIF = %q, want %q", inp.SARIF, "coverage.sarif")
		}
	})

	t.Run("sarif default empty", func(t *testing.T) {
		clear(t)
		t.Setenv("INPUT_FORMAT", "gocover")

		inp, err := ParseInputs()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inp.SARIF != "" {
			t.Errorf("SARIF = %q, want empty", inp.SARIF)
		}
	})
}

func TestFormatOrderAndParsersInSync(t *testing.T) {
	// Verify the invariant that init() checks
	if len(formatOrder) != len(parsers) {
		t.Errorf("formatOrder has %d entries, parsers has %d", len(formatOrder), len(parsers))
	}
	for _, f := range formatOrder {
		if _, ok := parsers[f]; !ok {
			t.Errorf("formatOrder contains %q but parsers does not", f)
		}
	}
}

