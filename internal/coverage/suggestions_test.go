package coverage

import (
	"strings"
	"testing"
)

func TestRankSuggestions(t *testing.T) {
	t.Run("ranks by uncovered lines descending", func(t *testing.T) {
		files := []FileCoverage{
			{Path: "a.go", Line: &Metric{Hit: 8, Total: 10}},   // 2 uncovered
			{Path: "b.go", Line: &Metric{Hit: 5, Total: 20}},   // 15 uncovered
			{Path: "c.go", Line: &Metric{Hit: 90, Total: 100}}, // 10 uncovered
		}

		suggestions := RankSuggestions(files)
		if len(suggestions) != 3 {
			t.Fatalf("expected 3 suggestions, got %d", len(suggestions))
		}
		if suggestions[0].Path != "b.go" {
			t.Errorf("expected b.go first, got %s", suggestions[0].Path)
		}
		if suggestions[1].Path != "c.go" {
			t.Errorf("expected c.go second, got %s", suggestions[1].Path)
		}
		if suggestions[2].Path != "a.go" {
			t.Errorf("expected a.go third, got %s", suggestions[2].Path)
		}
	})

	t.Run("skips fully covered files", func(t *testing.T) {
		files := []FileCoverage{
			{Path: "full.go", Line: &Metric{Hit: 10, Total: 10}},
			{Path: "partial.go", Line: &Metric{Hit: 5, Total: 10}},
		}

		suggestions := RankSuggestions(files)
		if len(suggestions) != 1 {
			t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
		}
		if suggestions[0].Path != "partial.go" {
			t.Errorf("expected partial.go, got %s", suggestions[0].Path)
		}
	})

	t.Run("limits to maxSuggestions", func(t *testing.T) {
		var files []FileCoverage
		for i := 0; i < 10; i++ {
			files = append(files, FileCoverage{
				Path: "file.go",
				Line: &Metric{Hit: 5, Total: 10},
			})
		}

		suggestions := RankSuggestions(files)
		if len(suggestions) != maxSuggestions {
			t.Errorf("expected %d suggestions, got %d", maxSuggestions, len(suggestions))
		}
	})

	t.Run("replaces smallest when heap is full", func(t *testing.T) {
		// Fill heap with 5 small-uncovered files, then add a bigger one
		var files []FileCoverage
		for i := 0; i < maxSuggestions; i++ {
			files = append(files, FileCoverage{
				Path: "small.go",
				Line: &Metric{Hit: 9, Total: 10}, // 1 uncovered
			})
		}
		// This should replace one of the small entries
		files = append(files, FileCoverage{
			Path: "big.go",
			Line: &Metric{Hit: 0, Total: 100}, // 100 uncovered
		})

		suggestions := RankSuggestions(files)
		if len(suggestions) != maxSuggestions {
			t.Fatalf("expected %d suggestions, got %d", maxSuggestions, len(suggestions))
		}
		// big.go should be first (most uncovered)
		if suggestions[0].Path != "big.go" {
			t.Errorf("expected big.go first, got %s", suggestions[0].Path)
		}
	})

	t.Run("skips files with zero total lines", func(t *testing.T) {
		files := []FileCoverage{
			{Path: "zero.go", Line: &Metric{Hit: 0, Total: 0}},
			{Path: "has.go", Line: &Metric{Hit: 5, Total: 10}},
		}
		suggestions := RankSuggestions(files)
		if len(suggestions) != 1 {
			t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
		}
		if suggestions[0].Path != "has.go" {
			t.Errorf("expected has.go, got %s", suggestions[0].Path)
		}
	})

	t.Run("empty files returns empty", func(t *testing.T) {
		suggestions := RankSuggestions(nil)
		if len(suggestions) != 0 {
			t.Errorf("expected 0 suggestions, got %d", len(suggestions))
		}
	})

	t.Run("skips files with nil line metric", func(t *testing.T) {
		files := []FileCoverage{
			{Path: "no_lines.go", Line: nil},
			{Path: "has_lines.go", Line: &Metric{Hit: 5, Total: 10}},
		}

		suggestions := RankSuggestions(files)
		if len(suggestions) != 1 {
			t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
		}
	})
}

func TestFormatSuggestions(t *testing.T) {
	t.Run("renders markdown table", func(t *testing.T) {
		suggestions := []Suggestion{
			{Path: "big.go", UncoveredLines: 50, TotalLines: 100, LinePct: 50.0},
			{Path: "small.go", UncoveredLines: 5, TotalLines: 20, LinePct: 75.0},
		}

		output := FormatSuggestions(suggestions)
		if !strings.Contains(output, "Top Opportunities") {
			t.Error("should contain header")
		}
		if !strings.Contains(output, "`big.go`") {
			t.Error("should contain big.go")
		}
		if !strings.Contains(output, "50.0%") {
			t.Error("should contain percentage")
		}
		if !strings.Contains(output, "50") {
			t.Error("should contain uncovered count")
		}
	})

	t.Run("empty suggestions returns empty", func(t *testing.T) {
		output := FormatSuggestions(nil)
		if output != "" {
			t.Errorf("expected empty string, got %q", output)
		}
	})

	t.Run("sanitizes paths with special characters", func(t *testing.T) {
		suggestions := []Suggestion{
			{Path: "file|with|pipes.go", UncoveredLines: 10, TotalLines: 20, LinePct: 50.0},
			{Path: "file`with`backticks.go", UncoveredLines: 5, TotalLines: 10, LinePct: 50.0},
			{Path: "file\nwith\nnewlines.go", UncoveredLines: 3, TotalLines: 10, LinePct: 70.0},
		}

		output := FormatSuggestions(suggestions)
		if strings.Contains(output, "|with|") {
			t.Error("pipes should be escaped")
		}
		if strings.Contains(output, "`with`") {
			t.Error("backticks should be stripped")
		}
	})
}
