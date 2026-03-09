package coverage

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

type cloverCoverage struct {
	XMLName xml.Name      `xml:"coverage"`
	Project cloverProject `xml:"project"`
}

type cloverProject struct {
	Metrics  cloverMetrics `xml:"metrics"`
	Packages []cloverPkg   `xml:"package"`
	Files    []cloverFile  `xml:"file"`
}

type cloverPkg struct {
	Files []cloverFile `xml:"file"`
}

type cloverFile struct {
	Name    string        `xml:"name,attr"`
	Path    string        `xml:"path,attr"`
	Metrics cloverMetrics `xml:"metrics"`
	Lines   []cloverLine  `xml:"line"`
}

type cloverLine struct {
	Num   int    `xml:"num,attr"`
	Type  string `xml:"type,attr"` // "stmt", "cond", "method"
	Count int64  `xml:"count,attr"`
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

	if m.Statements == 0 {
		return nil, fmt.Errorf("clover: no statement data found (is this a valid clover report?)")
	}

	// Collect all files from packages and top-level
	var allFiles []cloverFile
	for _, pkg := range cov.Project.Packages {
		allFiles = append(allFiles, pkg.Files...)
	}
	allFiles = append(allFiles, cov.Project.Files...)

	// Build per-file detail from line elements
	fileDetails := map[string]*FileLineDetail{}

	for _, f := range allFiles {
		name := f.Path
		if name == "" {
			name = f.Name
		}
		if name == "" {
			continue
		}

		detail := &FileLineDetail{
			Lines:     map[int]int64{},
			Branches:  map[string]int64{},
			Functions: map[string]int64{},
		}

		hasLineData := false
		for _, line := range f.Lines {
			hasLineData = true
			switch line.Type {
			case "stmt":
				if line.Count > detail.Lines[line.Num] {
					detail.Lines[line.Num] = line.Count
				}
			case "cond":
				// Track as both a line and a branch point
				if line.Count > detail.Lines[line.Num] {
					detail.Lines[line.Num] = line.Count
				}
				branchKey := strconv.Itoa(line.Num)
				if line.Count > detail.Branches[branchKey] {
					detail.Branches[branchKey] = line.Count
				}
			case "method":
				if line.Count > detail.Lines[line.Num] {
					detail.Lines[line.Num] = line.Count
				}
				funcKey := name + ":" + strconv.Itoa(line.Num)
				if line.Count > detail.Functions[funcKey] {
					detail.Functions[funcKey] = line.Count
				}
			}
		}

		if hasLineData {
			fileDetails[name] = detail
		}
	}

	// If we have line-level detail, compute summaries from it for merge support.
	// But also preserve the project-level metrics for single-file accuracy,
	// since project metrics may include data not in individual file line elements.
	result := &CoverageResult{
		Line:        &Metric{Hit: m.CoveredStatements, Total: m.Statements},
		FileDetails: fileDetails,
	}

	if m.Conditionals > 0 {
		result.Branch = &Metric{Hit: m.CoveredConditionals, Total: m.Conditionals}
	}

	if m.Methods > 0 {
		result.Function = &Metric{Hit: m.CoveredMethods, Total: m.Methods}
	}

	// Compute per-file summaries for suggestions
	for _, f := range allFiles {
		name := f.Path
		if name == "" {
			name = f.Name
		}
		if name == "" || f.Metrics.Statements == 0 {
			continue
		}

		fc := FileCoverage{
			Path: name,
			Line: &Metric{Hit: f.Metrics.CoveredStatements, Total: f.Metrics.Statements},
		}
		if f.Metrics.Conditionals > 0 {
			fc.Branch = &Metric{Hit: f.Metrics.CoveredConditionals, Total: f.Metrics.Conditionals}
		}
		if f.Metrics.Methods > 0 {
			fc.Function = &Metric{Hit: f.Metrics.CoveredMethods, Total: f.Metrics.Methods}
		}
		result.Files = append(result.Files, fc)
	}

	return result, nil
}
