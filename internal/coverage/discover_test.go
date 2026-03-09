package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverReports(t *testing.T) {
	t.Run("finds lcov.info in coverage dir", func(t *testing.T) {
		dir := t.TempDir()
		coverageDir := filepath.Join(dir, "coverage")
		if err := os.Mkdir(coverageDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(coverageDir, "lcov.info"), []byte("SF:foo\nend_of_record\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths, err := DiscoverReports("lcov", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 1 || paths[0] != "coverage/lcov.info" {
			t.Errorf("expected ['coverage/lcov.info'], got %v", paths)
		}
	})

	t.Run("finds cover.out for gocover", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "cover.out"), []byte("mode: set\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths, err := DiscoverReports("gocover", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 1 || paths[0] != "cover.out" {
			t.Errorf("expected ['cover.out'], got %v", paths)
		}
	})

	t.Run("returns error when no file found", func(t *testing.T) {
		dir := t.TempDir()

		_, err := DiscoverReports("lcov", dir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "auto-discovery") {
			t.Errorf("error should mention auto-discovery: %v", err)
		}
	})

	t.Run("returns error for unknown format", func(t *testing.T) {
		_, err := DiscoverReports("unknown", ".")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no default paths") {
			t.Errorf("error should mention no default paths: %v", err)
		}
	})

	t.Run("finds multiple matching paths", func(t *testing.T) {
		dir := t.TempDir()
		// Create both coverage/lcov.info and lcov.info — should find both
		coverageDir := filepath.Join(dir, "coverage")
		if err := os.Mkdir(coverageDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(coverageDir, "lcov.info"), []byte("SF:foo\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lcov.info"), []byte("SF:bar\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths, err := DiscoverReports("lcov", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 2 {
			t.Errorf("expected 2 paths, got %d: %v", len(paths), paths)
		}
		// First should be coverage/lcov.info (ordered by defaultPaths)
		if paths[0] != "coverage/lcov.info" {
			t.Errorf("expected first path 'coverage/lcov.info', got %q", paths[0])
		}
	})
}

func TestResolvePaths(t *testing.T) {
	t.Run("resolves single literal path", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "cover.out"), []byte("mode: set\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths, err := ResolvePaths("cover.out", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 1 || paths[0] != "cover.out" {
			t.Errorf("expected ['cover.out'], got %v", paths)
		}
	})

	t.Run("resolves comma-separated paths", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "unit.out"), []byte("mode: set\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "integration.out"), []byte("mode: set\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths, err := ResolvePaths("unit.out, integration.out", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 2 {
			t.Errorf("expected 2 paths, got %d: %v", len(paths), paths)
		}
	})

	t.Run("resolves glob pattern", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "unit.out"), []byte("mode: set\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "integration.out"), []byte("mode: set\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths, err := ResolvePaths("*.out", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 2 {
			t.Errorf("expected 2 paths, got %d: %v", len(paths), paths)
		}
	})

	t.Run("deduplicates paths", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "cover.out"), []byte("mode: set\n"), 0644); err != nil {
			t.Fatal(err)
		}

		paths, err := ResolvePaths("cover.out, cover.out", dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(paths) != 1 {
			t.Errorf("expected 1 path (deduplicated), got %d: %v", len(paths), paths)
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		dir := t.TempDir()

		_, err := ResolvePaths("nonexistent.out", dir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error should mention not found: %v", err)
		}
	})
}
