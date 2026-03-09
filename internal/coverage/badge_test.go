package coverage

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestBadgeColor(t *testing.T) {
	tests := []struct {
		pct  float64
		want string
	}{
		{95.0, "brightgreen"},
		{90.0, "brightgreen"},
		{85.0, "green"},
		{80.0, "green"},
		{75.0, "yellowgreen"},
		{65.0, "yellow"},
		{55.0, "orange"},
		{40.0, "red"},
		{0.0, "red"},
	}
	for _, tt := range tests {
		got := badgeColor(tt.pct)
		if got != tt.want {
			t.Errorf("badgeColor(%.1f) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

func TestBadgeHandlesNaN(t *testing.T) {
	svg := GenerateBadgeSVG(math.NaN())
	if !strings.Contains(svg, "0.0%") {
		t.Error("NaN should be clamped to 0.0%")
	}

	jsonStr := GenerateBadgeJSON(math.Inf(1))
	if !strings.Contains(jsonStr, "0.0%") {
		t.Error("Inf should be clamped to 0.0%")
	}
}

func TestGenerateBadgeJSON(t *testing.T) {
	result := GenerateBadgeJSON(82.5)

	var endpoint shieldsEndpoint
	if err := json.Unmarshal([]byte(result), &endpoint); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if endpoint.SchemaVersion != 1 {
		t.Errorf("schemaVersion = %d, want 1", endpoint.SchemaVersion)
	}
	if endpoint.Label != "coverage" {
		t.Errorf("label = %q, want %q", endpoint.Label, "coverage")
	}
	if endpoint.Message != "82.5%" {
		t.Errorf("message = %q, want %q", endpoint.Message, "82.5%")
	}
	if endpoint.Color != "green" {
		t.Errorf("color = %q, want %q", endpoint.Color, "green")
	}
}

func TestGenerateBadgeSVG(t *testing.T) {
	svg := GenerateBadgeSVG(82.5)

	if !strings.HasPrefix(svg, "<svg") {
		t.Error("SVG should start with <svg")
	}
	if !strings.HasSuffix(svg, "</svg>") {
		t.Error("SVG should end with </svg>")
	}
	if !strings.Contains(svg, "coverage") {
		t.Error("SVG should contain label 'coverage'")
	}
	if !strings.Contains(svg, "82.5%") {
		t.Error("SVG should contain value '82.5%'")
	}
	if !strings.Contains(svg, "#97ca00") {
		t.Error("SVG should contain green color for 82.5%")
	}
}
