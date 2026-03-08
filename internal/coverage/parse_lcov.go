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

	var lineFnd, lineHit int64
	var branchFnd, branchHit int64
	var funcFnd, funcHit int64
	var hasBranch, hasFunc bool
	var hasRecords bool

	// Per-file tracking
	var currentFile string
	var fileLine, fileBranch, fileFunc *Metric
	var files []FileCoverage

	flushFile := func() {
		if currentFile == "" {
			return
		}
		fc := FileCoverage{Path: currentFile, Line: fileLine, Branch: fileBranch, Function: fileFunc}
		files = append(files, fc)
		fileLine = nil
		fileBranch = nil
		fileFunc = nil
		currentFile = ""
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
		case "LF":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing LF value %q: %w", val, err)
			}
			lineFnd += n
			hasRecords = true
			if fileLine == nil {
				fileLine = &Metric{}
			}
			fileLine.Total = n
		case "LH":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing LH value %q: %w", val, err)
			}
			lineHit += n
			if fileLine == nil {
				fileLine = &Metric{}
			}
			fileLine.Hit = n
		case "BRF":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing BRF value %q: %w", val, err)
			}
			branchFnd += n
			hasBranch = true
			if fileBranch == nil {
				fileBranch = &Metric{}
			}
			fileBranch.Total = n
		case "BRH":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing BRH value %q: %w", val, err)
			}
			branchHit += n
			if fileBranch == nil {
				fileBranch = &Metric{}
			}
			fileBranch.Hit = n
		case "FNF":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing FNF value %q: %w", val, err)
			}
			funcFnd += n
			hasFunc = true
			if fileFunc == nil {
				fileFunc = &Metric{}
			}
			fileFunc.Total = n
		case "FNH":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing FNH value %q: %w", val, err)
			}
			funcHit += n
			if fileFunc == nil {
				fileFunc = &Metric{}
			}
			fileFunc.Hit = n
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

	result := &CoverageResult{
		Line:  &Metric{Hit: lineHit, Total: lineFnd},
		Files: files,
	}
	if hasBranch && branchFnd > 0 {
		result.Branch = &Metric{Hit: branchHit, Total: branchFnd}
	}
	if hasFunc && funcFnd > 0 {
		result.Function = &Metric{Hit: funcHit, Total: funcFnd}
	}

	return result, nil
}
