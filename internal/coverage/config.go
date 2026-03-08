package coverage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var validFormats = map[string]bool{
	"lcov":      true,
	"gocover":   true,
	"cobertura": true,
	"clover":    true,
	"jacoco":    true,
}

// LoadConfig reads and validates a coverage.json config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg.Version == 0 {
		return fmt.Errorf("config validation: version is required")
	}

	if len(cfg.Coverage) == 0 {
		return fmt.Errorf("config validation: coverage array must not be empty")
	}

	for i, entry := range cfg.Coverage {
		if strings.TrimSpace(entry.Name) == "" {
			return fmt.Errorf("config validation: coverage[%d].name is required", i)
		}
		if strings.TrimSpace(entry.Path) == "" {
			return fmt.Errorf("config validation: coverage[%d].path is required", i)
		}
		if strings.TrimSpace(entry.Format) == "" {
			return fmt.Errorf("config validation: coverage[%d].format is required", i)
		}
		if !validFormats[entry.Format] {
			return fmt.Errorf("config validation: coverage[%d].format %q is not a valid format (valid: lcov, gocover, cobertura, clover, jacoco)", i, entry.Format)
		}
		if entry.Threshold.Line == nil && entry.Threshold.Branch == nil && entry.Threshold.Function == nil {
			return fmt.Errorf("config validation: coverage[%d].threshold must set at least one of line, branch, or function", i)
		}
	}

	return nil
}
