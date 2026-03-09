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

	var hasBlocks bool

	// Block-level tracking for merge support
	blockDetails := map[string]map[string]*BlockEntry{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		// Format: file:start.col,end.col stmts count
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
		blockRef := rest[:secondLastSpace]

		stmts, err := strconv.ParseInt(stmtsStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing statement count %q: %w", stmtsStr, err)
		}

		count, err := strconv.ParseInt(countStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing execution count %q: %w", countStr, err)
		}

		// Extract file path and block key
		colonIdx := strings.LastIndex(blockRef, ":")
		filePath := blockRef
		blockKey := blockRef
		if colonIdx > 0 {
			filePath = blockRef[:colonIdx]
			blockKey = blockRef[colonIdx+1:]
		}

		if blockDetails[filePath] == nil {
			blockDetails[filePath] = map[string]*BlockEntry{}
		}

		if existing, ok := blockDetails[filePath][blockKey]; ok {
			// Same block seen multiple times — take max count
			if count > existing.Count {
				existing.Count = count
			}
		} else {
			blockDetails[filePath][blockKey] = &BlockEntry{Stmts: stmts, Count: count}
		}
		hasBlocks = true
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading gocover data: %w", err)
	}

	if !hasBlocks {
		return nil, fmt.Errorf("gocover: no coverage blocks found")
	}

	return computeBlockBasedSummary(blockDetails), nil
}
