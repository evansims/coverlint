package coverage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		fixture   string
		wantErr   bool
		wantCount int
	}{
		{
			name:      "valid config",
			fixture:   "valid_config.json",
			wantCount: 2,
		},
		{
			name:      "minimal config",
			fixture:   "minimal_config.json",
			wantCount: 1,
		},
		{
			name:    "nonexistent file",
			fixture: "nonexistent.json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", tt.fixture)
			cfg, err := LoadConfig(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if len(cfg.Coverage) != tt.wantCount {
					t.Errorf("got %d coverage entries, want %d", len(cfg.Coverage), tt.wantCount)
				}
				if cfg.Version != 1 {
					t.Errorf("got version %d, want 1", cfg.Version)
				}
			}
		})
	}
}

func TestLoadConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{
			name:    "missing version",
			json:    `{"coverage":[{"name":"x","path":"x","format":"lcov","threshold":{"line":80}}]}`,
			wantErr: "version",
		},
		{
			name:    "empty coverage array",
			json:    `{"version":1,"coverage":[]}`,
			wantErr: "coverage",
		},
		{
			name:    "missing name",
			json:    `{"version":1,"coverage":[{"path":"x","format":"lcov","threshold":{"line":80}}]}`,
			wantErr: "name",
		},
		{
			name:    "missing path",
			json:    `{"version":1,"coverage":[{"name":"x","format":"lcov","threshold":{"line":80}}]}`,
			wantErr: "path",
		},
		{
			name:    "missing format",
			json:    `{"version":1,"coverage":[{"name":"x","path":"x","threshold":{"line":80}}]}`,
			wantErr: "format",
		},
		{
			name:    "invalid format",
			json:    `{"version":1,"coverage":[{"name":"x","path":"x","format":"invalid","threshold":{"line":80}}]}`,
			wantErr: "format",
		},
		{
			name:    "no thresholds set",
			json:    `{"version":1,"coverage":[{"name":"x","path":"x","format":"lcov","threshold":{}}]}`,
			wantErr: "threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "coverage.json")
			if err := os.WriteFile(path, []byte(tt.json), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should mention %q", err.Error(), tt.wantErr)
			}
		})
	}
}
