package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

func setInputEnv(t *testing.T, env map[string]string) {
	t.Helper()
	// Clear all input env vars first
	for _, key := range []string{
		"INPUT_PATH", "INPUT_FORMAT",
		"INPUT_WORKING-DIRECTORY", "INPUT_FAIL-ON-ERROR",
		"INPUT_THRESHOLD-LINE", "INPUT_THRESHOLD-BRANCH", "INPUT_THRESHOLD-FUNCTION",
		"INPUT_SUGGESTIONS",
	} {
		t.Setenv(key, "")
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func TestRunIntegration(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	summaryFile := filepath.Join(t.TempDir(), "github_summary")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_FAIL-ON-ERROR":     "true",
		"INPUT_THRESHOLD-LINE":    "50",
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
	outputFile := filepath.Join(t.TempDir(), "github_output")
	summaryFile := filepath.Join(t.TempDir(), "github_summary")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_FAIL-ON-ERROR":     "true",
		"INPUT_THRESHOLD-LINE":    "80",
	})

	err := Run()
	if err == nil {
		t.Fatal("expected Run() to return error when threshold not met")
	}
}

func TestRunFailOnErrorFalse(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "github_output")
	summaryFile := filepath.Join(t.TempDir(), "github_summary")
	if err := os.WriteFile(outputFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(summaryFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_OUTPUT", outputFile)
	t.Setenv("GITHUB_STEP_SUMMARY", summaryFile)

	fixtureDir := filepath.Join("..", "..", "testdata", "lcov")

	setInputEnv(t, map[string]string{
		"INPUT_PATH":              "basic.info",
		"INPUT_FORMAT":            "lcov",
		"INPUT_WORKING-DIRECTORY": fixtureDir,
		"INPUT_FAIL-ON-ERROR":     "false",
		"INPUT_THRESHOLD-LINE":    "80",
	})

	// Should NOT error even though threshold fails, because fail-on-error is false
	if err := Run(); err != nil {
		t.Fatalf("Run() should not error with fail-on-error=false, got: %v", err)
	}
}
