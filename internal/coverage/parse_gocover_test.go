package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGocover(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantLine *Metric
		wantErr  bool
	}{
		{
			name:     "basic set mode",
			fixture:  "gocover/basic.out",
			wantLine: &Metric{Hit: 4, Total: 6},
		},
		{
			name:     "multi package count mode",
			fixture:  "gocover/multi_package.out",
			wantLine: &Metric{Hit: 5, Total: 7},
		},
		{
			name:    "empty profile",
			fixture: "gocover/empty.out",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "testdata", tt.fixture))
			if err != nil {
				t.Fatal(err)
			}
			result, err := parseGocover(data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseGocover() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			assertMetric(t, "line", result.Line, tt.wantLine)
			if result.Branch != nil {
				t.Errorf("branch: expected nil for gocover, got %+v", result.Branch)
			}
			if result.Function != nil {
				t.Errorf("function: expected nil for gocover, got %+v", result.Function)
			}
		})
	}
}

func TestParseGocoverBlockDetails(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gocover", "basic.out"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := parseGocover(data)
	if err != nil {
		t.Fatal(err)
	}

	if result.BlockDetails == nil {
		t.Fatal("expected BlockDetails to be populated")
	}
	if len(result.BlockDetails) == 0 {
		t.Fatal("expected at least one file in BlockDetails")
	}
}

func TestParseGocoverInvalidStmtCount(t *testing.T) {
	data := []byte("mode: set\nfoo.go:1.1,5.1 abc 1\n")
	_, err := parseGocover(data)
	if err == nil {
		t.Fatal("expected error for invalid statement count")
	}
	if !strings.Contains(err.Error(), "parsing statement count") {
		t.Errorf("error should mention statement count: %v", err)
	}
}

func TestParseGocoverInvalidExecCount(t *testing.T) {
	data := []byte("mode: set\nfoo.go:1.1,5.1 3 xyz\n")
	_, err := parseGocover(data)
	if err == nil {
		t.Fatal("expected error for invalid execution count")
	}
	if !strings.Contains(err.Error(), "parsing execution count") {
		t.Errorf("error should mention execution count: %v", err)
	}
}

func TestParseGocoverMalformedLines(t *testing.T) {
	// Lines with insufficient fields should be skipped
	data := []byte("mode: set\nfoo.go:1.1,5.1 3 1\nsingletoken\ntwotokens only\n")
	result, err := parseGocover(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Line == nil {
		t.Fatal("expected line metric from valid block")
	}
}

func TestParseGocoverDuplicateBlocks(t *testing.T) {
	// Same block with different counts — should take max
	data := []byte("mode: count\nfoo.go:1.1,5.1 3 2\nfoo.go:1.1,5.1 3 5\n")
	result, err := parseGocover(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 3 statements, count=max(2,5)=5, all covered
	if result.Line.Hit != 3 || result.Line.Total != 3 {
		t.Errorf("line: got {%d/%d}, want {3/3}", result.Line.Hit, result.Line.Total)
	}
}

func TestParseGocoverNoBlocks(t *testing.T) {
	data := []byte("mode: set\n")
	_, err := parseGocover(data)
	if err == nil {
		t.Fatal("expected error for no blocks")
	}
	if !strings.Contains(err.Error(), "no coverage blocks") {
		t.Errorf("error should mention no blocks: %v", err)
	}
}
