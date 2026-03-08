package coverage

// Threshold defines coverage percentage thresholds.
type Threshold struct {
	Line     *float64 `json:"line,omitempty"`
	Branch   *float64 `json:"branch,omitempty"`
	Function *float64 `json:"function,omitempty"`
}

// CoverageEntry defines a single coverage report to check.
type CoverageEntry struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Format    string    `json:"format"`
	Threshold Threshold `json:"threshold"`
}

// Config is the top-level coverage.json schema.
type Config struct {
	Version  int             `json:"version"`
	Coverage []CoverageEntry `json:"coverage"`
}

// CoverageResult holds parsed coverage metrics from a report.
type CoverageResult struct {
	Name     string
	Line     *Metric
	Branch   *Metric
	Function *Metric
}

// Metric holds hit/total counts for a coverage metric.
type Metric struct {
	Hit   int64
	Total int64
}

// Pct returns the coverage percentage, or 0 if total is 0.
func (m *Metric) Pct() float64 {
	if m.Total == 0 {
		return 0
	}
	return float64(m.Hit) / float64(m.Total) * 100
}

// Violation records a threshold that was not met.
type Violation struct {
	Entry    string
	Metric   string
	Actual   float64
	Required float64
}
