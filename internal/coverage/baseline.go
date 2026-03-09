package coverage

import (
	"encoding/json"
	"fmt"
	"time"
)

// BaselineData holds previous coverage data for delta comparison.
type BaselineData struct {
	Score    float64  `json:"score"`
	Line     *float64 `json:"line,omitempty"`
	Branch   *float64 `json:"branch,omitempty"`
	Function *float64 `json:"function,omitempty"`

	Timestamp string `json:"timestamp"`
}

// GenerateBaseline creates a BaselineData snapshot from the given results.
// It uses the last entry (the total/combined result).
func GenerateBaseline(results []EntryResult) BaselineData {
	bd := BaselineData{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if len(results) == 0 {
		return bd
	}

	last := results[len(results)-1]
	if last.Score != nil {
		bd.Score = *last.Score
	}
	bd.Line = last.Line
	bd.Branch = last.Branch
	bd.Function = last.Function

	return bd
}

// LoadBaseline parses baseline coverage data from a raw JSON string.
func LoadBaseline(source string) (*BaselineData, error) {
	if source == "" {
		return nil, fmt.Errorf("baseline JSON is empty")
	}

	var bd BaselineData
	if err := json.Unmarshal([]byte(source), &bd); err != nil {
		return nil, fmt.Errorf("parsing baseline JSON: %w", err)
	}

	return &bd, nil
}

// CompareBaseline checks whether the coverage delta meets the minimum allowed change.
// Returns a Violation if the score dropped more than allowed by minDelta.
func CompareBaseline(prev *BaselineData, currentScore float64, minDelta *float64) []Violation {
	if minDelta == nil {
		return nil
	}

	delta := currentScore - prev.Score
	if delta < *minDelta {
		return []Violation{
			{
				Entry:    "coverage",
				Metric:   "delta",
				Actual:   delta,
				Required: *minDelta,
			},
		}
	}

	return nil
}
