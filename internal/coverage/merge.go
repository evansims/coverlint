package coverage

// MergeResults combines multiple CoverageResults into one by merging
// at the line/block level. A line is considered covered if it was hit
// in ANY of the input reports. All results must be from the same format.
func MergeResults(results []*CoverageResult) *CoverageResult {
	if len(results) == 0 {
		return nil
	}
	if len(results) == 1 {
		return results[0]
	}

	// Determine merge strategy based on which detail is populated
	if results[0].BlockDetails != nil {
		return mergeBlockBased(results)
	}
	return mergeLineBased(results)
}

// mergeLineBased merges results from line-based formats (LCOV, Cobertura, Clover, JaCoCo).
func mergeLineBased(results []*CoverageResult) *CoverageResult {
	merged := map[string]*FileLineDetail{}

	for _, r := range results {
		for file, detail := range r.FileDetails {
			existing, ok := merged[file]
			if !ok {
				existing = &FileLineDetail{
					Lines:     map[int]int64{},
					Branches:  map[string]int64{},
					Functions: map[string]int64{},
				}
				merged[file] = existing
			}

			for line, count := range detail.Lines {
				if prev, exists := existing.Lines[line]; !exists || count > prev {
					existing.Lines[line] = count
				}
			}
			for key, count := range detail.Branches {
				if prev, exists := existing.Branches[key]; !exists || count > prev {
					existing.Branches[key] = count
				}
			}
			for name, count := range detail.Functions {
				if prev, exists := existing.Functions[name]; !exists || count > prev {
					existing.Functions[name] = count
				}
			}
		}
	}

	return computeLineBasedSummary(merged)
}

// mergeBlockBased merges results from gocover (block-based format).
func mergeBlockBased(results []*CoverageResult) *CoverageResult {
	merged := map[string]map[string]*BlockEntry{}

	for _, r := range results {
		for file, blocks := range r.BlockDetails {
			existing, ok := merged[file]
			if !ok {
				existing = map[string]*BlockEntry{}
				merged[file] = existing
			}

			for key, block := range blocks {
				if eb, ok := existing[key]; ok {
					if block.Count > eb.Count {
						eb.Count = block.Count
					}
				} else {
					existing[key] = &BlockEntry{Stmts: block.Stmts, Count: block.Count}
				}
			}
		}
	}

	return computeBlockBasedSummary(merged)
}

// computeLineBasedSummary recomputes summary metrics from merged line-level data.
func computeLineBasedSummary(merged map[string]*FileLineDetail) *CoverageResult {
	var totalLines, hitLines int64
	var totalBranches, hitBranches int64
	var totalFuncs, hitFuncs int64
	var hasBranches, hasFuncs bool

	var files []FileCoverage

	for path, detail := range merged {
		var fileLines, fileHitLines int64
		for _, count := range detail.Lines {
			totalLines++
			fileLines++
			if count > 0 {
				hitLines++
				fileHitLines++
			}
		}

		fc := FileCoverage{Path: path}
		if fileLines > 0 {
			fc.Line = &Metric{Hit: fileHitLines, Total: fileLines}
		}

		var fileBranches, fileHitBranches int64
		for _, count := range detail.Branches {
			totalBranches++
			fileBranches++
			hasBranches = true
			if count > 0 {
				hitBranches++
				fileHitBranches++
			}
		}
		if fileBranches > 0 {
			fc.Branch = &Metric{Hit: fileHitBranches, Total: fileBranches}
		}

		var fileFuncs, fileHitFuncs int64
		for _, count := range detail.Functions {
			totalFuncs++
			fileFuncs++
			hasFuncs = true
			if count > 0 {
				hitFuncs++
				fileHitFuncs++
			}
		}
		if fileFuncs > 0 {
			fc.Function = &Metric{Hit: fileHitFuncs, Total: fileFuncs}
		}

		files = append(files, fc)
	}

	result := &CoverageResult{
		Files:       files,
		FileDetails: merged,
	}

	if totalLines > 0 {
		result.Line = &Metric{Hit: hitLines, Total: totalLines}
	}
	if hasBranches && totalBranches > 0 {
		result.Branch = &Metric{Hit: hitBranches, Total: totalBranches}
	}
	if hasFuncs && totalFuncs > 0 {
		result.Function = &Metric{Hit: hitFuncs, Total: totalFuncs}
	}

	return result
}

// computeBlockBasedSummary recomputes summary metrics from merged gocover block data.
func computeBlockBasedSummary(merged map[string]map[string]*BlockEntry) *CoverageResult {
	var totalStmts, coveredStmts int64

	var files []FileCoverage

	for path, blocks := range merged {
		var fileTotal, fileCovered int64
		for _, block := range blocks {
			totalStmts += block.Stmts
			fileTotal += block.Stmts
			if block.Count > 0 {
				coveredStmts += block.Stmts
				fileCovered += block.Stmts
			}
		}
		files = append(files, FileCoverage{
			Path: path,
			Line: &Metric{Hit: fileCovered, Total: fileTotal},
		})
	}

	return &CoverageResult{
		Line:         &Metric{Hit: coveredStmts, Total: totalStmts},
		Files:        files,
		BlockDetails: merged,
	}
}
