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

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "end_of_record" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		val := parts[1]

		switch key {
		case "LF":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing LF value %q: %w", val, err)
			}
			lineFnd += n
			hasRecords = true
		case "LH":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing LH value %q: %w", val, err)
			}
			lineHit += n
		case "BRF":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing BRF value %q: %w", val, err)
			}
			branchFnd += n
			hasBranch = true
		case "BRH":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing BRH value %q: %w", val, err)
			}
			branchHit += n
		case "FNF":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing FNF value %q: %w", val, err)
			}
			funcFnd += n
			hasFunc = true
		case "FNH":
			n, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing FNH value %q: %w", val, err)
			}
			funcHit += n
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading lcov data: %w", err)
	}

	if !hasRecords {
		return nil, fmt.Errorf("lcov: no coverage records found")
	}

	result := &CoverageResult{
		Line: &Metric{Hit: lineHit, Total: lineFnd},
	}
	if hasBranch {
		result.Branch = &Metric{Hit: branchHit, Total: branchFnd}
	}
	if hasFunc {
		result.Function = &Metric{Hit: funcHit, Total: funcFnd}
	}

	return result, nil
}
