package coverage

import (
	"container/heap"
	"fmt"
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
// Uses a min-heap of size k for O(n log k) performance in large monorepos.
func RankSuggestions(files []FileCoverage) []Suggestion {
	var h suggestionHeap
	for _, f := range files {
		if f.Line == nil || f.Line.Total == 0 {
			continue
		}
		uncovered := f.Line.Total - f.Line.Hit
		if uncovered <= 0 {
			continue
		}
		s := Suggestion{
			Path:           f.Path,
			UncoveredLines: uncovered,
			TotalLines:     f.Line.Total,
			LinePct:        f.Line.Pct(),
		}
		if h.Len() < maxSuggestions {
			heap.Push(&h, s)
		} else if uncovered > h[0].UncoveredLines {
			h[0] = s
			heap.Fix(&h, 0)
		}
	}

	// Extract in descending order
	result := make([]Suggestion, h.Len())
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = heap.Pop(&h).(Suggestion)
	}
	return result
}

// suggestionHeap is a min-heap ordered by UncoveredLines (smallest at root).
type suggestionHeap []Suggestion

func (h suggestionHeap) Len() int            { return len(h) }
func (h suggestionHeap) Less(i, j int) bool  { return h[i].UncoveredLines < h[j].UncoveredLines }
func (h suggestionHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *suggestionHeap) Push(x any)         { *h = append(*h, x.(Suggestion)) }
func (h *suggestionHeap) Pop() any           { old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x }

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
