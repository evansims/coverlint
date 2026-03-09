package coverage

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func parseLcov(data []byte) (*CoverageResult, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	var hasRecords bool

	// Per-file detail tracking for merge support
	fileDetails := map[string]*FileLineDetail{}
	var currentFile string
	var currentDetail *FileLineDetail
	var hasDetailLines bool // tracks if any DA/BRDA/FNDA lines were found

	// Summary-line fallback tracking (for files without DA lines)
	type fileSummary struct {
		lf, lh, brf, brh, fnf, fnh int64
	}
	fileSummaries := map[string]*fileSummary{}
	var currentSummary *fileSummary

	ensureDetail := func() {
		if currentDetail != nil {
			return
		}
		currentDetail = &FileLineDetail{
			Lines:     map[int]int64{},
			Branches:  map[string]int64{},
			Functions: map[string]int64{},
		}
	}

	ensureSummary := func() {
		if currentSummary != nil {
			return
		}
		currentSummary = &fileSummary{}
	}

	flushFile := func() {
		if currentFile == "" {
			return
		}
		if currentDetail != nil {
			fileDetails[currentFile] = currentDetail
		}
		if currentSummary != nil {
			fileSummaries[currentFile] = currentSummary
		}
		currentFile = ""
		currentDetail = nil
		currentSummary = nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "end_of_record" {
			flushFile()
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		val := parts[1]

		switch key {
		case "SF":
			currentFile = val
			ensureDetail()
			ensureSummary()
			hasRecords = true

		case "DA":
			// DA:line_number,execution_count
			daParts := strings.SplitN(val, ",", 2)
			if len(daParts) != 2 {
				continue
			}
			lineNum, err := strconv.Atoi(daParts[0])
			if err != nil {
				continue
			}
			count, err := strconv.ParseInt(daParts[1], 10, 64)
			if err != nil {
				continue
			}
			ensureDetail()
			currentDetail.Lines[lineNum] = count
			hasDetailLines = true

		case "BRDA":
			// BRDA:line,block,branch,taken  (taken may be "-" for not executed)
			brdaParts := strings.SplitN(val, ",", 4)
			if len(brdaParts) != 4 {
				continue
			}
			taken := brdaParts[3]
			var count int64
			if taken != "-" {
				n, err := strconv.ParseInt(taken, 10, 64)
				if err != nil {
					continue
				}
				count = n
			}
			branchKey := brdaParts[0] + ":" + brdaParts[1] + ":" + brdaParts[2]
			ensureDetail()
			currentDetail.Branches[branchKey] = count
			hasDetailLines = true

		case "FNDA":
			// FNDA:execution_count,function_name
			fndaParts := strings.SplitN(val, ",", 2)
			if len(fndaParts) != 2 {
				continue
			}
			count, err := strconv.ParseInt(fndaParts[0], 10, 64)
			if err != nil {
				continue
			}
			ensureDetail()
			currentDetail.Functions[fndaParts[1]] = count
			hasDetailLines = true

		case "FN":
			// FN:line_number,function_name — defines a function (count from FNDA)
			fnParts := strings.SplitN(val, ",", 2)
			if len(fnParts) != 2 {
				continue
			}
			ensureDetail()
			// Register function with 0 count if not yet seen
			if _, ok := currentDetail.Functions[fnParts[1]]; !ok {
				currentDetail.Functions[fnParts[1]] = 0
			}

		// Summary lines — used as fallback when no DA lines present
		case "LF":
			n, _ := strconv.ParseInt(val, 10, 64)
			ensureSummary()
			currentSummary.lf = n
		case "LH":
			n, _ := strconv.ParseInt(val, 10, 64)
			ensureSummary()
			currentSummary.lh = n
		case "BRF":
			n, _ := strconv.ParseInt(val, 10, 64)
			ensureSummary()
			currentSummary.brf = n
		case "BRH":
			n, _ := strconv.ParseInt(val, 10, 64)
			ensureSummary()
			currentSummary.brh = n
		case "FNF":
			n, _ := strconv.ParseInt(val, 10, 64)
			ensureSummary()
			currentSummary.fnf = n
		case "FNH":
			n, _ := strconv.ParseInt(val, 10, 64)
			ensureSummary()
			currentSummary.fnh = n
		}
	}

	// Flush last file if no trailing end_of_record
	flushFile()

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading lcov data: %w", err)
	}

	if !hasRecords {
		return nil, fmt.Errorf("lcov: no coverage records found")
	}

	// If we have detail lines (DA/BRDA/FNDA), compute summaries from them
	if hasDetailLines {
		return computeLineBasedSummary(fileDetails), nil
	}

	// Fallback: use summary lines (LF/LH/BRF/BRH/FNF/FNH)
	var totalLines, hitLines int64
	var totalBranches, hitBranches int64
	var totalFuncs, hitFuncs int64
	var hasBranch, hasFunc bool

	for _, s := range fileSummaries {
		totalLines += s.lf
		hitLines += s.lh
		if s.brf > 0 {
			hasBranch = true
			totalBranches += s.brf
			hitBranches += s.brh
		}
		if s.fnf > 0 {
			hasFunc = true
			totalFuncs += s.fnf
			hitFuncs += s.fnh
		}
	}

	result := &CoverageResult{}
	if totalLines > 0 {
		result.Line = &Metric{Hit: hitLines, Total: totalLines}
	}
	if hasBranch && totalBranches > 0 {
		result.Branch = &Metric{Hit: hitBranches, Total: totalBranches}
	}
	if hasFunc && totalFuncs > 0 {
		result.Function = &Metric{Hit: hitFuncs, Total: totalFuncs}
	}

	return result, nil
}
