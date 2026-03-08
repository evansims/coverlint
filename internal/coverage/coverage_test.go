package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

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

	configDir := t.TempDir()

	// Copy lcov fixture
	lcovData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "lcov", "basic.info"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "lcov.info"), lcovData, 0644); err != nil {
		t.Fatal(err)
	}

	// Config with threshold below actual coverage (75% line, actual is 3/4 = 75%)
	configJSON := `{
		"version": 1,
		"coverage": [{
			"name": "test",
			"path": "lcov.info",
			"format": "lcov",
			"threshold": {"line": 50}
		}]
	}`
	if err := os.WriteFile(filepath.Join(configDir, "coverage.json"), []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("INPUT_CONFIG", "coverage.json")
	t.Setenv("INPUT_WORKING-DIRECTORY", configDir)
	t.Setenv("INPUT_FAIL-ON-ERROR", "true")

	err = Run()
	if err != nil {
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

	configDir := t.TempDir()

	lcovData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "lcov", "basic.info"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "lcov.info"), lcovData, 0644); err != nil {
		t.Fatal(err)
	}

	// Threshold higher than actual (75% actual, 80% required)
	configJSON := `{
		"version": 1,
		"coverage": [{
			"name": "test",
			"path": "lcov.info",
			"format": "lcov",
			"threshold": {"line": 80}
		}]
	}`
	if err := os.WriteFile(filepath.Join(configDir, "coverage.json"), []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("INPUT_CONFIG", "coverage.json")
	t.Setenv("INPUT_WORKING-DIRECTORY", configDir)
	t.Setenv("INPUT_FAIL-ON-ERROR", "true")

	err = Run()
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

	configDir := t.TempDir()

	lcovData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "lcov", "basic.info"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "lcov.info"), lcovData, 0644); err != nil {
		t.Fatal(err)
	}

	configJSON := `{
		"version": 1,
		"coverage": [{
			"name": "test",
			"path": "lcov.info",
			"format": "lcov",
			"threshold": {"line": 80}
		}]
	}`
	if err := os.WriteFile(filepath.Join(configDir, "coverage.json"), []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("INPUT_CONFIG", "coverage.json")
	t.Setenv("INPUT_WORKING-DIRECTORY", configDir)
	t.Setenv("INPUT_FAIL-ON-ERROR", "false")

	// Should NOT error even though threshold fails, because fail-on-error is false
	err = Run()
	if err != nil {
		t.Fatalf("Run() should not error with fail-on-error=false, got: %v", err)
	}
}
