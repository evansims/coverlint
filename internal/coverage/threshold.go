package coverage

import "fmt"

// CheckResult holds the outcome of threshold checking for a single entry.
type CheckResult struct {
	Passed     bool
	Violations []Violation
	Skipped    []SkippedThreshold
}

func CheckThresholds(result *CoverageResult, threshold *Threshold) CheckResult {
	var cr CheckResult

	checkMetric(&cr, "coverage", "line", result.Line, threshold.Line)
	checkMetric(&cr, "coverage", "branch", result.Branch, threshold.Branch)
	checkMetric(&cr, "coverage", "function", result.Function, threshold.Function)

	cr.Passed = len(cr.Violations) == 0
	return cr
}

func checkMetric(cr *CheckResult, entry, metric string, m *Metric, threshold *float64) {
	if threshold == nil {
		return
	}
	if m == nil {
		cr.Skipped = append(cr.Skipped, SkippedThreshold{Entry: entry, Metric: metric})
		return
	}

	pct := m.Pct()
	if pct < *threshold {
		cr.Violations = append(cr.Violations, Violation{
			Entry:    entry,
			Metric:   metric,
			Actual:   pct,
			Required: *threshold,
		})
	}
}

func FormatViolation(v Violation) string {
	return fmt.Sprintf("%s: %s coverage %.1f%% < %.1f%% required",
		v.Entry, v.Metric, v.Actual, v.Required)
}
