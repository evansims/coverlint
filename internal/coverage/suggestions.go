package coverage

import (
	"fmt"
	"sort"
	"strings"
)

const maxSuggestions = 5

// Suggestion represents a file with uncovered lines that could improve overall coverage.
type Suggestion struct {
	Path           string
	UncoveredLines int64
	TotalLines     int64
	LinePct        float64
}

// RankSuggestions returns the top files by number of uncovered lines,
// which represent the biggest opportunities for coverage improvement.
func RankSuggestions(files []FileCoverage) []Suggestion {
	var suggestions []Suggestion
	for _, f := range files {
		if f.Line == nil || f.Line.Total == 0 {
			continue
		}
		uncovered := f.Line.Total - f.Line.Hit
		if uncovered <= 0 {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Path:           f.Path,
			UncoveredLines: uncovered,
			TotalLines:     f.Line.Total,
			LinePct:        f.Line.Pct(),
		})
	}

	// Sort by uncovered lines descending (biggest impact first)
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].UncoveredLines > suggestions[j].UncoveredLines
	})

	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}
	return suggestions
}

// FormatSuggestions renders the suggestions as a markdown section for the job summary.
func FormatSuggestions(suggestions []Suggestion) string {
	if len(suggestions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### Top Opportunities for Coverage Improvement\n\n")
	sb.WriteString("| File | Coverage | Uncovered Lines |\n")
	sb.WriteString("|------|----------|----------------|\n")

	for _, s := range suggestions {
		safePath := strings.ReplaceAll(s.Path, "|", "\\|")
		safePath = strings.ReplaceAll(safePath, "`", "")
		safePath = strings.ReplaceAll(safePath, "\n", " ")
		fmt.Fprintf(&sb, "| `%s` | %.1f%% | %d |\n", safePath, s.LinePct, s.UncoveredLines)
	}
	sb.WriteString("\n")
	return sb.String()
}
