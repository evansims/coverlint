package coverage

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func parseGocover(data []byte) (*CoverageResult, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	var totalStmts, coveredStmts int64
	var hasBlocks bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		lastSpace := strings.LastIndex(line, " ")
		if lastSpace < 0 {
			continue
		}
		countStr := line[lastSpace+1:]
		rest := line[:lastSpace]

		secondLastSpace := strings.LastIndex(rest, " ")
		if secondLastSpace < 0 {
			continue
		}
		stmtsStr := rest[secondLastSpace+1:]

		stmts, err := strconv.ParseInt(stmtsStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing statement count %q: %w", stmtsStr, err)
		}

		count, err := strconv.ParseInt(countStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing execution count %q: %w", countStr, err)
		}

		totalStmts += stmts
		if count > 0 {
			coveredStmts += stmts
		}
		hasBlocks = true
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading gocover data: %w", err)
	}

	if !hasBlocks {
		return nil, fmt.Errorf("gocover: no coverage blocks found")
	}

	return &CoverageResult{
		Line: &Metric{Hit: coveredStmts, Total: totalStmts},
	}, nil
}
