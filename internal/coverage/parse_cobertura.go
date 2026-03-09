package coverage

import (
	"encoding/xml"
	"fmt"
)

type coberturaCoverage struct {
	XMLName         xml.Name          `xml:"coverage"`
	LinesCovered    int64             `xml:"lines-covered,attr"`
	LinesValid      int64             `xml:"lines-valid,attr"`
	BranchesCovered int64             `xml:"branches-covered,attr"`
	BranchesValid   int64             `xml:"branches-valid,attr"`
	Packages        []coberturaPackge `xml:"packages>package"`
}

type coberturaPackge struct {
	Classes []coberturaClass `xml:"classes>class"`
}

type coberturaClass struct {
	Filename string            `xml:"filename,attr"`
	Methods  []coberturaMethod `xml:"methods>method"`
	Lines    []coberturaLine   `xml:"lines>line"`
}

type coberturaMethod struct {
	Name  string          `xml:"name,attr"`
	Lines []coberturaLine `xml:"lines>line"`
}

type coberturaLine struct {
	Number int64 `xml:"number,attr"`
	Hits   int64 `xml:"hits,attr"`
}

func parseCobertura(data []byte) (*CoverageResult, error) {
	var cov coberturaCoverage
	if err := xml.Unmarshal(data, &cov); err != nil {
		return nil, fmt.Errorf("parsing cobertura XML: %w", err)
	}

	if cov.LinesValid == 0 {
		return nil, fmt.Errorf("cobertura: no line data found (is this a valid cobertura report?)")
	}

	fileDetails := map[string]*FileLineDetail{}

	for _, pkg := range cov.Packages {
		for _, cls := range pkg.Classes {
			detail, ok := fileDetails[cls.Filename]
			if !ok {
				detail = &FileLineDetail{
					Lines:     map[int]int64{},
					Branches:  map[string]int64{},
					Functions: map[string]int64{},
				}
				fileDetails[cls.Filename] = detail
			}

			// Track lines from class-level line elements
			for _, line := range cls.Lines {
				lineNum := int(line.Number)
				if line.Hits > detail.Lines[lineNum] {
					detail.Lines[lineNum] = line.Hits
				}
			}

			// Track methods and their lines
			for _, method := range cls.Methods {
				var methodHit bool
				for _, line := range method.Lines {
					lineNum := int(line.Number)
					if line.Hits > detail.Lines[lineNum] {
						detail.Lines[lineNum] = line.Hits
					}
					if line.Hits > 0 {
						methodHit = true
					}
				}
				// Use class+method as key to handle same method name in different classes
				funcKey := cls.Filename + ":" + method.Name
				if methodHit {
					detail.Functions[funcKey] = 1
				} else if _, exists := detail.Functions[funcKey]; !exists {
					detail.Functions[funcKey] = 0
				}
			}
		}
	}

	// If parsers populated line-level detail, use it for summary.
	// But Cobertura also provides top-level summary attributes which
	// may account for lines not in <class> elements (e.g., Python coverage.py
	// doesn't emit <method> elements). Use the top-level summary for
	// Line/Branch metrics, and compute Function from detail.
	result := &CoverageResult{
		Line:        &Metric{Hit: cov.LinesCovered, Total: cov.LinesValid},
		FileDetails: fileDetails,
	}

	if cov.BranchesValid > 0 {
		result.Branch = &Metric{Hit: cov.BranchesCovered, Total: cov.BranchesValid}
	}

	// Compute function coverage from detail
	var totalFuncs, coveredFuncs int64
	for _, detail := range fileDetails {
		for _, count := range detail.Functions {
			totalFuncs++
			if count > 0 {
				coveredFuncs++
			}
		}
	}
	if totalFuncs > 0 {
		result.Function = &Metric{Hit: coveredFuncs, Total: totalFuncs}
	}

	// Compute per-file summaries for suggestions
	for path, detail := range fileDetails {
		fc := FileCoverage{Path: path}
		if len(detail.Lines) > 0 {
			var hit, total int64
			for _, count := range detail.Lines {
				total++
				if count > 0 {
					hit++
				}
			}
			fc.Line = &Metric{Hit: hit, Total: total}
		}
		if len(detail.Functions) > 0 {
			var hit, total int64
			for _, count := range detail.Functions {
				total++
				if count > 0 {
					hit++
				}
			}
			fc.Function = &Metric{Hit: hit, Total: total}
		}
		result.Files = append(result.Files, fc)
	}

	return result, nil
}
