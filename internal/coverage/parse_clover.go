package coverage

import (
	"encoding/xml"
	"fmt"
)

type cloverCoverage struct {
	XMLName xml.Name      `xml:"coverage"`
	Project cloverProject `xml:"project"`
}

type cloverProject struct {
	Metrics cloverMetrics `xml:"metrics"`
}

type cloverMetrics struct {
	Statements          int64 `xml:"statements,attr"`
	CoveredStatements   int64 `xml:"coveredstatements,attr"`
	Conditionals        int64 `xml:"conditionals,attr"`
	CoveredConditionals int64 `xml:"coveredconditionals,attr"`
	Methods             int64 `xml:"methods,attr"`
	CoveredMethods      int64 `xml:"coveredmethods,attr"`
}

func parseClover(data []byte) (*CoverageResult, error) {
	var cov cloverCoverage
	if err := xml.Unmarshal(data, &cov); err != nil {
		return nil, fmt.Errorf("parsing clover XML: %w", err)
	}

	m := cov.Project.Metrics

	result := &CoverageResult{
		Line: &Metric{Hit: m.CoveredStatements, Total: m.Statements},
	}

	if m.Conditionals > 0 {
		result.Branch = &Metric{Hit: m.CoveredConditionals, Total: m.Conditionals}
	}

	if m.Methods > 0 {
		result.Function = &Metric{Hit: m.CoveredMethods, Total: m.Methods}
	}

	return result, nil
}
