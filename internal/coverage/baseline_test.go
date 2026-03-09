package coverage

import (
	"encoding/json"
	"testing"
)

func TestGenerateBaseline(t *testing.T) {
	t.Run("from results uses last entry", func(t *testing.T) {
		score1 := 70.0
		line1 := 75.0
		score2 := 85.0
		line2 := 90.0
		branch2 := 72.0

		results := []EntryResult{
			{Name: "gocover", Score: &score1, Line: &line1},
			{Name: "Total", Score: &score2, Line: &line2, Branch: &branch2},
		}

		bd := GenerateBaseline(results)
		if bd.Score != 85.0 {
			t.Errorf("Score = %v, want 85.0", bd.Score)
		}
		if bd.Line == nil || *bd.Line != 90.0 {
			t.Errorf("Line = %v, want 90.0", bd.Line)
		}
		if bd.Branch == nil || *bd.Branch != 72.0 {
			t.Errorf("Branch = %v, want 72.0", bd.Branch)
		}
		if bd.Function != nil {
			t.Errorf("Function = %v, want nil", bd.Function)
		}
		if bd.Timestamp == "" {
			t.Error("Timestamp should not be empty")
		}
	})

	t.Run("empty results", func(t *testing.T) {
		bd := GenerateBaseline(nil)
		if bd.Score != 0 {
			t.Errorf("Score = %v, want 0", bd.Score)
		}
		if bd.Timestamp == "" {
			t.Error("Timestamp should not be empty")
		}
	})
}

func TestLoadBaseline(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		input := `{"score":85.5,"line":90.0,"branch":72.0,"timestamp":"2025-01-01T00:00:00Z"}`

		bd, err := LoadBaseline(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bd.Score != 85.5 {
			t.Errorf("Score = %v, want 85.5", bd.Score)
		}
		if bd.Line == nil || *bd.Line != 90.0 {
			t.Errorf("Line = %v, want 90.0", bd.Line)
		}
		if bd.Branch == nil || *bd.Branch != 72.0 {
			t.Errorf("Branch = %v, want 72.0", bd.Branch)
		}
		if bd.Timestamp != "2025-01-01T00:00:00Z" {
			t.Errorf("Timestamp = %q, want 2025-01-01T00:00:00Z", bd.Timestamp)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := LoadBaseline("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := LoadBaseline("not json")
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestCompareBaseline(t *testing.T) {
	t.Run("no regression", func(t *testing.T) {
		prev := &BaselineData{Score: 80.0}
		minDelta := 0.0
		violations := CompareBaseline(prev, 85.0, &minDelta)
		if len(violations) != 0 {
			t.Errorf("expected no violations, got %d", len(violations))
		}
	})

	t.Run("regression detected", func(t *testing.T) {
		prev := &BaselineData{Score: 80.0}
		minDelta := 0.0
		violations := CompareBaseline(prev, 75.0, &minDelta)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(violations))
		}
		if violations[0].Metric != "delta" {
			t.Errorf("Metric = %q, want 'delta'", violations[0].Metric)
		}
		if violations[0].Actual != -5.0 {
			t.Errorf("Actual = %v, want -5.0", violations[0].Actual)
		}
	})

	t.Run("allowed drop within threshold", func(t *testing.T) {
		prev := &BaselineData{Score: 80.0}
		minDelta := -5.0
		violations := CompareBaseline(prev, 75.0, &minDelta)
		if len(violations) != 0 {
			t.Errorf("expected no violations for drop within threshold, got %d", len(violations))
		}
	})

	t.Run("exceeds allowed drop", func(t *testing.T) {
		prev := &BaselineData{Score: 80.0}
		minDelta := -2.0
		violations := CompareBaseline(prev, 75.0, &minDelta)
		if len(violations) != 1 {
			t.Fatalf("expected 1 violation, got %d", len(violations))
		}
		if violations[0].Actual != -5.0 {
			t.Errorf("Actual = %v, want -5.0", violations[0].Actual)
		}
		if violations[0].Required != -2.0 {
			t.Errorf("Required = %v, want -2.0", violations[0].Required)
		}
	})

	t.Run("nil min-delta skips comparison", func(t *testing.T) {
		prev := &BaselineData{Score: 80.0}
		violations := CompareBaseline(prev, 75.0, nil)
		if violations != nil {
			t.Errorf("expected nil violations when min-delta is nil, got %v", violations)
		}
	})
}

func TestBaselineDataJSON(t *testing.T) {
	line := 90.0
	bd := BaselineData{
		Score:     85.0,
		Line:      &line,
		Timestamp: "2025-01-01T00:00:00Z",
	}

	data, err := json.Marshal(bd)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed BaselineData
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.Score != bd.Score {
		t.Errorf("Score = %v, want %v", parsed.Score, bd.Score)
	}
	if parsed.Line == nil || *parsed.Line != *bd.Line {
		t.Errorf("Line = %v, want %v", parsed.Line, bd.Line)
	}
	if parsed.Branch != nil {
		t.Error("Branch should be nil (omitted)")
	}
}
