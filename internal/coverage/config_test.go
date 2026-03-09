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
				"INPUT_THRESHOLD-LINE": "80",
			},
			wantFormats: []string{"gocover"},
		},
		{
			name: "valid all thresholds",
			env: map[string]string{
				"INPUT_PATH":               "lcov.info",
				"INPUT_FORMAT":             "lcov",
				"INPUT_NAME":               "backend",
				"INPUT_THRESHOLD-LINE":     "80",
				"INPUT_THRESHOLD-BRANCH":   "70",
				"INPUT_THRESHOLD-FUNCTION": "75",
			},
			wantFormats: []string{"lcov"},
		},
		{
			name: "path optional",
			env: map[string]string{
				"INPUT_FORMAT":         "lcov",
				"INPUT_THRESHOLD-LINE": "80",
			},
			wantFormats: []string{"lcov"},
		},
		{
			name: "multiple formats",
			env: map[string]string{
				"INPUT_FORMAT":         "gocover,lcov",
				"INPUT_THRESHOLD-LINE": "80",
			},
			wantFormats: []string{"gocover", "lcov"},
		},
		{
			name: "multiple formats with spaces",
			env: map[string]string{
				"INPUT_FORMAT":         "gocover, lcov, cobertura",
				"INPUT_THRESHOLD-LINE": "80",
			},
			wantFormats: []string{"gocover", "lcov", "cobertura"},
		},
		{
			name: "missing format",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_THRESHOLD-LINE": "80",
			},
			wantErr: "format is required",
		},
		{
			name: "invalid format",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "invalid",
				"INPUT_THRESHOLD-LINE": "80",
			},
			wantErr: "not valid",
		},
		{
			name: "one invalid in multi-format",
			env: map[string]string{
				"INPUT_FORMAT":         "gocover,invalid",
				"INPUT_THRESHOLD-LINE": "80",
			},
			wantErr: "not valid",
		},
		{
			name: "no thresholds",
			env: map[string]string{
				"INPUT_PATH":   "cover.out",
				"INPUT_FORMAT": "gocover",
			},
			wantErr: "at least one threshold",
		},
		{
			name: "negative threshold",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "lcov",
				"INPUT_THRESHOLD-LINE": "-5",
			},
			wantErr: "between 0 and 100",
		},
		{
			name: "threshold over 100",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "lcov",
				"INPUT_THRESHOLD-LINE": "200",
			},
			wantErr: "between 0 and 100",
		},
		{
			name: "non-numeric threshold",
			env: map[string]string{
				"INPUT_PATH":           "cover.out",
				"INPUT_FORMAT":         "lcov",
				"INPUT_THRESHOLD-LINE": "abc",
			},
			wantErr: "not a valid number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all input env vars
			for _, key := range []string{
				"INPUT_PATH", "INPUT_FORMAT", "INPUT_NAME",
				"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
				"INPUT_THRESHOLD-LINE", "INPUT_THRESHOLD-BRANCH", "INPUT_THRESHOLD-FUNCTION",
				"INPUT_SUGGESTIONS",
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

func TestParseInputsDefaults(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT", "INPUT_NAME",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_THRESHOLD-LINE", "INPUT_THRESHOLD-BRANCH", "INPUT_THRESHOLD-FUNCTION",
	} {
		t.Setenv(key, "")
	}

	t.Setenv("INPUT_PATH", "cover.out")
	t.Setenv("INPUT_FORMAT", "gocover")
	t.Setenv("INPUT_THRESHOLD-LINE", "80")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inp.Name != "gocover" {
		t.Errorf("name should default to format, got %q", inp.Name)
	}
	if inp.WorkDir != "." {
		t.Errorf("workdir should default to '.', got %q", inp.WorkDir)
	}
	if !inp.FailOnError {
		t.Error("fail-on-error should default to true")
	}
}

func TestParseInputsMultiFormatName(t *testing.T) {
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT", "INPUT_NAME",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_THRESHOLD-LINE", "INPUT_THRESHOLD-BRANCH", "INPUT_THRESHOLD-FUNCTION",
	} {
		t.Setenv(key, "")
	}

	t.Setenv("INPUT_FORMAT", "gocover,lcov")
	t.Setenv("INPUT_THRESHOLD-LINE", "80")

	inp, err := ParseInputs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inp.Name != "gocover, lcov" {
		t.Errorf("name should default to joined formats, got %q", inp.Name)
	}
}
