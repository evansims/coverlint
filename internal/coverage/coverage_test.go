package coverage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setInputEnv(t *testing.T, env map[string]string) {
	t.Helper()
	// Clear all input env vars first
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_MIN-COVERAGE", "INPUT_MIN-LINE", "INPUT_MIN-BRANCH", "INPUT_MIN-FUNCTION",
		"INPUT_WEIGHT-LINE", "INPUT_WEIGHT-BRANCH", "INPUT_WEIGHT-FUNCTION",
		"INPUT_SUGGESTIONS", "INPUT_ANNOTATIONS",
	} {
		t.Setenv(key, "")
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func setupGitHubEnv(t *testing.T) (outputFile, summaryFile string) {
	t.Helper()
	outputFile = filepath.Join(t.TempDir(), "github_output")
	summaryFile = filepath.Join(t.TempDir(), "github_summary")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)
	return
}

func TestRunIntegration(t *testing.T) {
	outputFile, summaryFile := setupGitHubEnv(t)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_FAIL-ON-ERROR":     "true",
		"INPUT_MIN-LINE":          "50",
	})

	if err := Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	output, _ := os.ReadFile(outputFile)
	if len(output) == 0 {
		t.Error("expected outputs to be written")
	}

	summary, _ := os.ReadFile(summaryFile)
	if len(summary) == 0 {
		t.Error("expected summary to be written")
	}
}

func TestRunThresholdFailure(t *testing.T) {
	setupGitHubEnv(t)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_FAIL-ON-ERROR":     "true",
		"INPUT_MIN-LINE":          "80",
	})

	err := Run()
	if err == nil {
		t.Fatal("expected Run() to return error when threshold not met")
	}
}

func TestRunAutoFormat(t *testing.T) {
	outputFile, summaryFile := setupGitHubEnv(t)

	// Use gocover testdata — place a cover.out in a temp dir so auto-discovery finds it
	dir := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "gocover", "basic.out")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cover.out"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// No format, no path — both auto-discovered
	setInputEnv(t, map[string]string{
		"INPUT_WORKING-DIRECTORY": dir,
		"INPUT_MIN-LINE":          "50",
	})

	if err := Run(); err != nil {
		t.Fatalf("Run() with auto-format returned error: %v", err)
	}

	output, _ := os.ReadFile(outputFile)
	if len(output) == 0 {
		t.Error("expected outputs to be written")
	}

	summary, _ := os.ReadFile(summaryFile)
	if len(summary) == 0 {
		t.Error("expected summary to be written")
	}
}

func TestRunFailOnErrorFalse(t *testing.T) {
	setupGitHubEnv(t)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_FAIL-ON-ERROR":     "false",
		"INPUT_MIN-LINE":          "80",
	})

	// Should NOT error even though threshold fails, because fail-on-error is false
	if err := Run(); err != nil {
		t.Fatalf("Run() should not error with fail-on-error=false, got: %v", err)
	}
}

func TestRunExplicitFormatAutoDiscoverPaths(t *testing.T) {
	setupGitHubEnv(t)

	// Create a temp dir with a gocover file at a default path
	dir := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "gocover", "basic.out")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cover.out"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Explicit format, no path — should auto-discover paths for the format
	setInputEnv(t, map[string]string{
		"INPUT_FORMAT":            "gocover",
		"INPUT_WORKING-DIRECTORY": dir,
		"INPUT_MIN-LINE":          "50",
	})

	if err := Run(); err != nil {
		t.Fatalf("Run() with explicit format and auto-discover paths returned error: %v", err)
	}
}

func TestRunMultiFormat(t *testing.T) {
	outputFile, summaryFile := setupGitHubEnv(t)

	dir := t.TempDir()

	// Create gocover file
	gocoverData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gocover", "basic.out"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cover.out"), gocoverData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create lcov file
	lcovData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "lcov", "basic.info"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lcov.info"), lcovData, 0644); err != nil {
		t.Fatal(err)
	}

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "cover.out, lcov.info",
		"INPUT_FORMAT":            "gocover, lcov",
		"INPUT_WORKING-DIRECTORY": dir,
		"INPUT_MIN-LINE":          "50",
	})

	if err := Run(); err != nil {
		t.Fatalf("Run() multi-format returned error: %v", err)
	}

	output, _ := os.ReadFile(outputFile)
	if !strings.Contains(string(output), "passed=true") {
		t.Error("expected passed=true in output")
	}

	summary, _ := os.ReadFile(summaryFile)
	content := string(summary)
	// Multi-format should show Total row
	if !strings.Contains(content, "**Total**") {
		t.Error("expected bold Total row in multi-format summary")
	}
}

func TestRunWithSuggestions(t *testing.T) {
	_, summaryFile := setupGitHubEnv(t)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	// Use fixture with DA lines so per-file detail is available for suggestions
	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "with_suggestions.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_SUGGESTIONS":       "true",
	})

	if err := Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	summary, _ := os.ReadFile(summaryFile)
	content := string(summary)
	if !strings.Contains(content, "Top Opportunities") {
		t.Error("expected suggestions section in summary")
	}
}

func TestRunSuggestionsDisabled(t *testing.T) {
	_, summaryFile := setupGitHubEnv(t)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "multi_file.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_SUGGESTIONS":       "false",
	})

	if err := Run(); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	summary, _ := os.ReadFile(summaryFile)
	content := string(summary)
	if strings.Contains(content, "Top Opportunities") {
		t.Error("should not contain suggestions when disabled")
	}
}

func TestRunNoReportsParsed(t *testing.T) {
	setupGitHubEnv(t)

	dir := t.TempDir()
	// Create a file that is not a valid gocover file
	if err := os.WriteFile(filepath.Join(dir, "cover.out"), []byte("garbage"), 0644); err != nil {
		t.Fatal(err)
	}

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "cover.out",
		"INPUT_FORMAT":            "gocover",
		"INPUT_WORKING-DIRECTORY": dir,
	})

	err := Run()
	if err == nil {
		t.Fatal("expected error when no valid reports parsed")
	}
}

func TestRunSkippedThresholds(t *testing.T) {
	setupGitHubEnv(t)

	fixtureDir := filepath.Join("..", "..", "testdata", "gocover")

	// Gocover doesn't report branch/function, setting thresholds for them should skip
	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.out",
		"INPUT_FORMAT":            "gocover",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_MIN-BRANCH":        "50",
		"INPUT_MIN-FUNCTION":      "50",
	})

	// Should pass (skipped thresholds don't cause failures)
	if err := Run(); err != nil {
		t.Fatalf("Run() with skipped thresholds returned error: %v", err)
	}
}

func TestRunNoThresholds(t *testing.T) {
	setupGitHubEnv(t)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
	})

	if err := Run(); err != nil {
		t.Fatalf("Run() with no thresholds returned error: %v", err)
	}
}

func TestRunMinCoverageThreshold(t *testing.T) {
	setupGitHubEnv(t)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_MIN-COVERAGE":      "99",
		"INPUT_FAIL-ON-ERROR":     "true",
	})

	err := Run()
	if err == nil {
		t.Fatal("expected error when min-coverage threshold not met")
	}
}

func TestReadCoverageFile(t *testing.T) {
	t.Run("reads valid file", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "test.out")
		content := []byte("mode: set\nfoo.go:1.1,2.1 1 1\n")
		if err := os.WriteFile(f, content, 0644); err != nil {
			t.Fatal(err)
		}
		data, err := readCoverageFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("data mismatch")
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := readCoverageFile("/nonexistent/path/file.out")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("errors on oversized file", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "big.out")
		// Create a file that appears large via sparse file
		fh, err := os.Create(f)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = fh.Close() }()
		// Seek past max size to make a sparse file
		if _, err := fh.Seek(maxCoverageFileSize+1, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := fh.Write([]byte{0}); err != nil {
			t.Fatal(err)
		}

		_, err = readCoverageFile(f)
		if err == nil {
			t.Fatal("expected error for oversized file")
		}
		if !strings.Contains(err.Error(), "exceeds maximum size") {
			t.Errorf("error should mention size: %v", err)
		}
	})
}

func TestParseWithFormats(t *testing.T) {
	t.Run("auto-detect format from files", func(t *testing.T) {
		dir := t.TempDir()
		gocoverData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gocover", "basic.out"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cover.out"), gocoverData, 0644); err != nil {
			t.Fatal(err)
		}

		results, err := parseWithFormats([]string{"cover.out"}, formatOrder, dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 format result, got %d", len(results))
		}
		if results[0].Format != "gocover" {
			t.Errorf("expected gocover format, got %s", results[0].Format)
		}
	})

	t.Run("errors on unknown format", func(t *testing.T) {
		_, err := parseWithFormats([]string{"file.out"}, []string{"nonexistent"}, ".")
		if err == nil {
			t.Fatal("expected error for unknown format")
		}
	})

	t.Run("errors when no parser succeeds", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bad.out"), []byte("garbage"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := parseWithFormats([]string{"bad.out"}, []string{"gocover"}, dir)
		if err == nil {
			t.Fatal("expected error when no parser succeeds")
		}
		if !strings.Contains(err.Error(), "no configured parser succeeded") {
			t.Errorf("error should mention no parser succeeded: %v", err)
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := parseWithFormats([]string{"nonexistent.out"}, []string{"gocover"}, t.TempDir())
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestBuildEntryResult(t *testing.T) {
	t.Run("all metrics", func(t *testing.T) {
		r := &CoverageResult{
			Line:     &Metric{Hit: 80, Total: 100},
			Branch:   &Metric{Hit: 60, Total: 100},
			Function: &Metric{Hit: 90, Total: 100},
		}
		entry := buildEntryResult("test", r, DefaultWeights())
		if entry.Name != "test" {
			t.Errorf("name = %q, want %q", entry.Name, "test")
		}
		if entry.Line == nil || *entry.Line != 80.0 {
			t.Errorf("line = %v, want 80.0", entry.Line)
		}
		if entry.Branch == nil || *entry.Branch != 60.0 {
			t.Errorf("branch = %v, want 60.0", entry.Branch)
		}
		if entry.Function == nil || *entry.Function != 90.0 {
			t.Errorf("function = %v, want 90.0", entry.Function)
		}
		if entry.Score == nil {
			t.Fatal("expected score to be set")
		}
	})

	t.Run("line only", func(t *testing.T) {
		r := &CoverageResult{
			Line: &Metric{Hit: 80, Total: 100},
		}
		entry := buildEntryResult("lineonly", r, DefaultWeights())
		if entry.Branch != nil {
			t.Error("expected nil branch")
		}
		if entry.Function != nil {
			t.Error("expected nil function")
		}
	})

	t.Run("no metrics", func(t *testing.T) {
		r := &CoverageResult{}
		entry := buildEntryResult("empty", r, DefaultWeights())
		if entry.Line != nil {
			t.Error("expected nil line")
		}
		if entry.Score == nil || *entry.Score != 0 {
			t.Errorf("expected score 0, got %v", entry.Score)
		}
	})
}

func TestDiscoverAndParse(t *testing.T) {
	t.Run("discovers and parses gocover", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join("..", "..", "testdata", "gocover", "basic.out")
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cover.out"), data, 0644); err != nil {
			t.Fatal(err)
		}

		results, err := discoverAndParse("gocover", dir, NewAnnotator(AnnotationConfig{Mode: "none"}, io.Discard))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Line == nil {
			t.Error("expected line metric")
		}
	})

	t.Run("errors for unknown format", func(t *testing.T) {
		_, err := discoverAndParse("nonexistent", ".", NewAnnotator(AnnotationConfig{Mode: "none"}, io.Discard))
		if err == nil {
			t.Fatal("expected error for unknown format")
		}
	})

	t.Run("errors when no files discovered", func(t *testing.T) {
		dir := t.TempDir()
		_, err := discoverAndParse("gocover", dir, NewAnnotator(AnnotationConfig{Mode: "none"}, io.Discard))
		if err == nil {
			t.Fatal("expected error when no files discovered")
		}
	})

	t.Run("errors when discovered file fails to parse", func(t *testing.T) {
		dir := t.TempDir()
		// Create a file at the default path but with invalid content
		if err := os.WriteFile(filepath.Join(dir, "cover.out"), []byte("garbage content"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := discoverAndParse("gocover", dir, NewAnnotator(AnnotationConfig{Mode: "none"}, io.Discard))
		if err == nil {
			t.Fatal("expected error when file fails to parse")
		}
	})

	t.Run("discovers and parses lcov", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join("..", "..", "testdata", "lcov", "basic.info")
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lcov.info"), data, 0644); err != nil {
			t.Fatal(err)
		}

		results, err := discoverAndParse("lcov", dir, NewAnnotator(AnnotationConfig{Mode: "none"}, io.Discard))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
	})
}
