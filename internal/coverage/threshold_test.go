package coverage

import "testing"

func floatPtr(f float64) *float64 { return &f }

func TestCheckThresholds(t *testing.T) {
	tests := []struct {
		name           string
		result         CoverageResult
		threshold      Threshold
		wantPassed     bool
		wantViolations int
		wantSkipped    int
	}{
		{
			name: "all pass",
			result: CoverageResult{
				Line: &Metric{Hit: 85, Total: 100},
			},
			threshold:      Threshold{Line: floatPtr(80)},
			wantPassed:     true,
			wantViolations: 0,
		},
		{
			name: "line fails",
			result: CoverageResult{
				Line: &Metric{Hit: 70, Total: 100},
			},
			threshold:      Threshold{Line: floatPtr(80)},
			wantPassed:     false,
			wantViolations: 1,
		},
		{
			name: "multiple failures",
			result: CoverageResult{
				Line:     &Metric{Hit: 70, Total: 100},
				Branch:   &Metric{Hit: 50, Total: 100},
				Function: &Metric{Hit: 60, Total: 100},
			},
			threshold:      Threshold{Line: floatPtr(80), Branch: floatPtr(70), Function: floatPtr(80)},
			wantPassed:     false,
			wantViolations: 3,
		},
		{
			name: "metric nil but threshold set reports skipped",
			result: CoverageResult{
				Line: &Metric{Hit: 90, Total: 100},
			},
			threshold:      Threshold{Line: floatPtr(80), Branch: floatPtr(70)},
			wantPassed:     true,
			wantViolations: 0,
			wantSkipped:    1,
		},
		{
			name: "exactly at threshold passes",
			result: CoverageResult{
				Line: &Metric{Hit: 80, Total: 100},
			},
			threshold:      Threshold{Line: floatPtr(80)},
			wantPassed:     true,
			wantViolations: 0,
		},
		{
			name: "multiple skipped thresholds",
			result: CoverageResult{
				Line: &Metric{Hit: 90, Total: 100},
				// Branch and Function nil — e.g. gocover format
			},
			threshold:      Threshold{Line: floatPtr(80), Branch: floatPtr(70), Function: floatPtr(80)},
			wantPassed:     true,
			wantViolations: 0,
			wantSkipped:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := CheckThresholds(&tt.result, &tt.threshold)
			if cr.Passed != tt.wantPassed {
				t.Errorf("passed = %v, want %v", cr.Passed, tt.wantPassed)
			}
			if len(cr.Violations) != tt.wantViolations {
				t.Errorf("got %d violations, want %d: %+v", len(cr.Violations), tt.wantViolations, cr.Violations)
			}
			if len(cr.Skipped) != tt.wantSkipped {
				t.Errorf("got %d skipped, want %d: %+v", len(cr.Skipped), tt.wantSkipped, cr.Skipped)
			}
		})
	}
}
