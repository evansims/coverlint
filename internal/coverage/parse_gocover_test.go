package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGocover(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantLine *Metric
		wantErr  bool
	}{
		{
			name:    "basic set mode",
			fixture: "gocover/basic.out",
			wantLine: &Metric{Hit: 4, Total: 6},
		},
		{
			name:    "multi package count mode",
			fixture: "gocover/multi_package.out",
			wantLine: &Metric{Hit: 5, Total: 7},
		},
		{
			name:    "empty profile",
			fixture: "gocover/empty.out",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", tt.fixture))
			if err != nil {
				t.Fatal(err)
			}
			result, err := parseGocover(data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseGocover() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			assertMetric(t, "line", result.Line, tt.wantLine)
			if result.Branch != nil {
				t.Errorf("branch: expected nil for gocover, got %+v", result.Branch)
			}
			if result.Function != nil {
				t.Errorf("function: expected nil for gocover, got %+v", result.Function)
			}
		})
	}
}
