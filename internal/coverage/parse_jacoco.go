package coverage

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

type jacocoReport struct {
	XMLName  xml.Name        `xml:"report"`
	Counters []jacocoCounter `xml:"counter"`
	Packages []jacocoPackage `xml:"package"`
}

type jacocoPackage struct {
	Name        string             `xml:"name,attr"`
	SourceFiles []jacocoSourceFile `xml:"sourcefile"`
}

type jacocoSourceFile struct {
	Name     string          `xml:"name,attr"`
	Lines    []jacocoLine    `xml:"line"`
	Counters []jacocoCounter `xml:"counter"`
}

type jacocoLine struct {
	Nr int   `xml:"nr,attr"`
	Mi int64 `xml:"mi,attr"` // missed instructions
	Ci int64 `xml:"ci,attr"` // covered instructions
	Mb int64 `xml:"mb,attr"` // missed branches
	Cb int64 `xml:"cb,attr"` // covered branches
}

type jacocoCounter struct {
	Type    string `xml:"type,attr"`
	Missed  int64  `xml:"missed,attr"`
	Covered int64  `xml:"covered,attr"`
}

func parseJacoco(data []byte) (*CoverageResult, error) {
	var report jacocoReport
	if err := xml.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parsing jacoco XML: %w", err)
	}

	if len(report.Counters) == 0 {
		return nil, fmt.Errorf("jacoco: no counters found at report level")
	}

	// Use report-level counters for single-file summary
	result := &CoverageResult{}

	for _, c := range report.Counters {
		total := c.Missed + c.Covered
		switch c.Type {
		case "LINE":
			result.Line = &Metric{Hit: c.Covered, Total: total}
		case "BRANCH":
			if total > 0 {
				result.Branch = &Metric{Hit: c.Covered, Total: total}
			}
		case "METHOD":
			if total > 0 {
				result.Function = &Metric{Hit: c.Covered, Total: total}
			}
		}
	}

	if result.Line == nil {
		return nil, fmt.Errorf("jacoco: no LINE counter found")
	}

	// Build per-file detail from line elements and sourcefile counters
	fileDetails := map[string]*FileLineDetail{}

	for _, pkg := range report.Packages {
		for _, sf := range pkg.SourceFiles {
			filePath := pkg.Name + "/" + sf.Name
			detail := &FileLineDetail{
				Lines:     map[int]int64{},
				Branches:  map[string]int64{},
				Functions: map[string]int64{},
			}

			for _, line := range sf.Lines {
				// A line is covered if it has any covered instructions
				if line.Ci > 0 {
					detail.Lines[line.Nr] = line.Ci
				} else {
					detail.Lines[line.Nr] = 0
				}

				// Track individual branch points per line
				totalBranches := line.Mb + line.Cb
				if totalBranches > 0 {
					for i := int64(0); i < line.Cb; i++ {
						detail.Branches[strconv.Itoa(line.Nr)+":"+strconv.FormatInt(i, 10)] = 1
					}
					for i := int64(0); i < line.Mb; i++ {
						detail.Branches[strconv.Itoa(line.Nr)+":missed:"+strconv.FormatInt(i, 10)] = 0
					}
				}
			}

			// Track methods from sourcefile-level METHOD counter
			for _, c := range sf.Counters {
				if c.Type == "METHOD" {
					// We don't have individual method names from line-level data,
					// so track covered/missed method counts using synthetic keys
					for i := int64(0); i < c.Covered; i++ {
						detail.Functions[filePath+":method:"+strconv.FormatInt(i, 10)] = 1
					}
					for i := int64(0); i < c.Missed; i++ {
						detail.Functions[filePath+":method:missed:"+strconv.FormatInt(i, 10)] = 0
					}
				}
			}

			if len(detail.Lines) > 0 {
				fileDetails[filePath] = detail
			}

			// Build FileCoverage for suggestions
			fc := FileCoverage{Path: filePath}
			for _, c := range sf.Counters {
				total := c.Missed + c.Covered
				switch c.Type {
				case "LINE":
					fc.Line = &Metric{Hit: c.Covered, Total: total}
				case "BRANCH":
					if total > 0 {
						fc.Branch = &Metric{Hit: c.Covered, Total: total}
					}
				case "METHOD":
					if total > 0 {
						fc.Function = &Metric{Hit: c.Covered, Total: total}
					}
				}
			}
			if fc.Line != nil {
				result.Files = append(result.Files, fc)
			}
		}
	}

	result.FileDetails = fileDetails

	return result, nil
}
