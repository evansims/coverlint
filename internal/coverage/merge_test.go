package coverage

import (
	"testing"
)

func TestMergeResults(t *testing.T) {
	t.Run("nil for empty input", func(t *testing.T) {
		result := MergeResults(nil)
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})

	t.Run("returns single result unchanged", func(t *testing.T) {
		r := &CoverageResult{
			Line: &Metric{Hit: 5, Total: 10},
		}
		result := MergeResults([]*CoverageResult{r})
		if result != r {
			t.Error("expected same pointer for single result")
		}
	})

	t.Run("merges line-based results with deduplication", func(t *testing.T) {
		r1 := &CoverageResult{
			FileDetails: map[string]*FileLineDetail{
				"foo.go": {
					Lines:     map[int]int64{1: 1, 2: 0, 3: 1},
					Branches:  map[string]int64{},
					Functions: map[string]int64{"foo": 1},
				},
			},
		}
		r2 := &CoverageResult{
			FileDetails: map[string]*FileLineDetail{
				"foo.go": {
					Lines:     map[int]int64{1: 0, 2: 1, 3: 1},
					Branches:  map[string]int64{},
					Functions: map[string]int64{"foo": 1, "bar": 0},
				},
			},
		}

		result := MergeResults([]*CoverageResult{r1, r2})
		if result.Line == nil {
			t.Fatal("expected Line metric")
		}
		// Lines: 1→max(1,0)=1, 2→max(0,1)=1, 3→max(1,1)=1 → 3 hit / 3 total
		if result.Line.Hit != 3 || result.Line.Total != 3 {
			t.Errorf("line: got {%d/%d}, want {3/3}", result.Line.Hit, result.Line.Total)
		}
		// Functions: foo→max(1,1)=1, bar→0 → 1 hit / 2 total
		if result.Function == nil {
			t.Fatal("expected Function metric")
		}
		if result.Function.Hit != 1 || result.Function.Total != 2 {
			t.Errorf("function: got {%d/%d}, want {1/2}", result.Function.Hit, result.Function.Total)
		}
	})

	t.Run("merges different files from different reports", func(t *testing.T) {
		r1 := &CoverageResult{
			FileDetails: map[string]*FileLineDetail{
				"a.go": {
					Lines:     map[int]int64{1: 1, 2: 1},
					Branches:  map[string]int64{},
					Functions: map[string]int64{},
				},
			},
		}
		r2 := &CoverageResult{
			FileDetails: map[string]*FileLineDetail{
				"b.go": {
					Lines:     map[int]int64{1: 0, 2: 1, 3: 1},
					Branches:  map[string]int64{},
					Functions: map[string]int64{},
				},
			},
		}

		result := MergeResults([]*CoverageResult{r1, r2})
		// a.go: 2/2, b.go: 2/3 → total 4/5
		if result.Line.Hit != 4 || result.Line.Total != 5 {
			t.Errorf("line: got {%d/%d}, want {4/5}", result.Line.Hit, result.Line.Total)
		}
	})

	t.Run("merges block-based results", func(t *testing.T) {
		r1 := &CoverageResult{
			BlockDetails: map[string]map[string]*BlockEntry{
				"foo.go": {
					"1.1,5.1": {Stmts: 3, Count: 1},
					"6.1,10.1": {Stmts: 2, Count: 0},
				},
			},
		}
		r2 := &CoverageResult{
			BlockDetails: map[string]map[string]*BlockEntry{
				"foo.go": {
					"1.1,5.1": {Stmts: 3, Count: 0},
					"6.1,10.1": {Stmts: 2, Count: 1},
				},
			},
		}

		result := MergeResults([]*CoverageResult{r1, r2})
		if result.Line == nil {
			t.Fatal("expected Line metric")
		}
		// Block 1: stmts=3, max(1,0)=1 → covered
		// Block 2: stmts=2, max(0,1)=1 → covered
		// Total: 5 covered / 5 total
		if result.Line.Hit != 5 || result.Line.Total != 5 {
			t.Errorf("line: got {%d/%d}, want {5/5}", result.Line.Hit, result.Line.Total)
		}
	})

	t.Run("merges branch data with max", func(t *testing.T) {
		r1 := &CoverageResult{
			FileDetails: map[string]*FileLineDetail{
				"foo.go": {
					Lines:     map[int]int64{1: 1},
					Branches:  map[string]int64{"1:0:0": 1, "1:0:1": 0},
					Functions: map[string]int64{},
				},
			},
		}
		r2 := &CoverageResult{
			FileDetails: map[string]*FileLineDetail{
				"foo.go": {
					Lines:     map[int]int64{1: 1},
					Branches:  map[string]int64{"1:0:0": 0, "1:0:1": 1},
					Functions: map[string]int64{},
				},
			},
		}

		result := MergeResults([]*CoverageResult{r1, r2})
		if result.Branch == nil {
			t.Fatal("expected Branch metric")
		}
		// Branch 1:0:0 → max(1,0)=1, 1:0:1 → max(0,1)=1 → 2/2
		if result.Branch.Hit != 2 || result.Branch.Total != 2 {
			t.Errorf("branch: got {%d/%d}, want {2/2}", result.Branch.Hit, result.Branch.Total)
		}
	})

	t.Run("merges mixed format results (line-based + block-based)", func(t *testing.T) {
		// Simulates monorepo: Go project (block-based) + Node project (line-based)
		// Detail data must be consistent since merge recomputes from details.
		goResult := &CoverageResult{
			BlockDetails: map[string]map[string]*BlockEntry{
				"main.go": {
					"1.1,5.1":  {Stmts: 5, Count: 1}, // 5 covered
					"6.1,10.1": {Stmts: 3, Count: 0}, // 0 covered
				},
			},
		}
		nodeResult := &CoverageResult{
			FileDetails: map[string]*FileLineDetail{
				"index.ts": {
					Lines:     map[int]int64{1: 1, 2: 1, 3: 0, 4: 1},
					Branches:  map[string]int64{"1:0:0": 1, "1:0:1": 0},
					Functions: map[string]int64{"main": 1, "helper": 0},
				},
			},
		}

		result := MergeResults([]*CoverageResult{goResult, nodeResult})

		// Go: 5 covered / 8 total stmts. Node: 3/4 lines hit.
		// Combined: (5+3)=8 hit / (8+4)=12 total
		if result.Line == nil {
			t.Fatal("expected Line metric")
		}
		if result.Line.Hit != 8 || result.Line.Total != 12 {
			t.Errorf("line: got {%d/%d}, want {8/12}", result.Line.Hit, result.Line.Total)
		}

		// Branch: only from node = 1/2
		if result.Branch == nil {
			t.Fatal("expected Branch metric")
		}
		if result.Branch.Hit != 1 || result.Branch.Total != 2 {
			t.Errorf("branch: got {%d/%d}, want {1/2}", result.Branch.Hit, result.Branch.Total)
		}

		// Function: only from node = 1/2
		if result.Function == nil {
			t.Fatal("expected Function metric")
		}
		if result.Function.Hit != 1 || result.Function.Total != 2 {
			t.Errorf("function: got {%d/%d}, want {1/2}", result.Function.Hit, result.Function.Total)
		}

		// Files: should have both
		if len(result.Files) != 2 {
			t.Errorf("expected 2 files, got %d", len(result.Files))
		}
	})
}
