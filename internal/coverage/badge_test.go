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
		{70.0, "yellowgreen"},
		{65.0, "yellow"},
		{60.0, "yellow"},
		{55.0, "orange"},
		{50.0, "orange"},
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

func TestBadgeColorHex(t *testing.T) {
	tests := []struct {
		pct  float64
		want string
	}{
		{95.0, "#4c1"},
		{90.0, "#4c1"},
		{85.0, "#97ca00"},
		{80.0, "#97ca00"},
		{75.0, "#a4a61d"},
		{70.0, "#a4a61d"},
		{65.0, "#dfb317"},
		{60.0, "#dfb317"},
		{55.0, "#fe7d37"},
		{50.0, "#fe7d37"},
		{40.0, "#e05d44"},
		{0.0, "#e05d44"},
	}
	for _, tt := range tests {
		got := badgeColorHex(tt.pct)
		if got != tt.want {
			t.Errorf("badgeColorHex(%.1f) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}

func TestBadgeHandlesNaN(t *testing.T) {
	svg := GenerateBadgeSVG(math.NaN())
	if !strings.Contains(svg, "0%") {
		t.Error("NaN should be clamped to 0%")
	}

	jsonStr := GenerateBadgeJSON(math.Inf(1))
	if !strings.Contains(jsonStr, "0%") {
		t.Error("Inf should be clamped to 0%")
	}
}

func TestClampPct(t *testing.T) {
	tests := []struct {
		name string
		pct  float64
		want float64
	}{
		{"normal", 50.0, 50.0},
		{"NaN", math.NaN(), 0},
		{"positive inf", math.Inf(1), 0},
		{"negative inf", math.Inf(-1), 0},
		{"zero", 0, 0},
		{"100", 100, 100},
	}
	for _, tt := range tests {
		got := clampPct(tt.pct)
		if got != tt.want {
			t.Errorf("clampPct(%v) = %v, want %v", tt.pct, got, tt.want)
		}
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
	if endpoint.Message != "82%" {
		t.Errorf("message = %q, want %q", endpoint.Message, "82%")
	}
	if endpoint.Color != "green" {
		t.Errorf("color = %q, want %q", endpoint.Color, "green")
	}
}

func TestGenerateBadgeJSONRoundsToWholeNumber(t *testing.T) {
	tests := []struct {
		pct     float64
		wantMsg string
	}{
		{0.0, "0%"},
		{50.4, "50%"},
		{50.5, "50%"}, // banker's rounding: rounds to even
		{50.6, "51%"},
		{99.9, "100%"},
		{100.0, "100%"},
	}
	for _, tt := range tests {
		result := GenerateBadgeJSON(tt.pct)
		var endpoint shieldsEndpoint
		if err := json.Unmarshal([]byte(result), &endpoint); err != nil {
			t.Fatalf("invalid JSON for pct %.1f: %v", tt.pct, err)
		}
		if endpoint.Message != tt.wantMsg {
			t.Errorf("pct %.1f: message = %q, want %q", tt.pct, endpoint.Message, tt.wantMsg)
		}
	}
}

func TestGenerateBadgeJSONAtBoundaries(t *testing.T) {
	tests := []struct {
		pct       float64
		wantColor string
	}{
		{0.0, "red"},
		{50.0, "orange"},
		{60.0, "yellow"},
		{70.0, "yellowgreen"},
		{80.0, "green"},
		{90.0, "brightgreen"},
		{100.0, "brightgreen"},
	}
	for _, tt := range tests {
		result := GenerateBadgeJSON(tt.pct)
		var endpoint shieldsEndpoint
		if err := json.Unmarshal([]byte(result), &endpoint); err != nil {
			t.Fatalf("invalid JSON for pct %.1f: %v", tt.pct, err)
		}
		if endpoint.Color != tt.wantColor {
			t.Errorf("pct %.1f: color = %q, want %q", tt.pct, endpoint.Color, tt.wantColor)
		}
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
	// 82.5 rounds to 82% (banker's rounding)
	if !strings.Contains(svg, "82%") {
		t.Error("SVG should contain rounded value '82%'")
	}
	if !strings.Contains(svg, "#97ca00") {
		t.Error("SVG should contain green color for 82.5%")
	}
}

func TestGenerateBadgeSVGRoundsToWholeNumber(t *testing.T) {
	tests := []struct {
		pct     float64
		wantVal string
	}{
		{0.0, "0%"},
		{75.3, "75%"},
		{75.7, "76%"},
		{100.0, "100%"},
	}
	for _, tt := range tests {
		svg := GenerateBadgeSVG(tt.pct)
		if !strings.Contains(svg, tt.wantVal) {
			t.Errorf("SVG for %.1f%% should contain %q", tt.pct, tt.wantVal)
		}
	}
}

func TestGenerateBadgeSVGColors(t *testing.T) {
	tests := []struct {
		pct       float64
		wantColor string
	}{
		{0.0, "#e05d44"},
		{55.0, "#fe7d37"},
		{65.0, "#dfb317"},
		{75.0, "#a4a61d"},
		{85.0, "#97ca00"},
		{95.0, "#4c1"},
	}
	for _, tt := range tests {
		svg := GenerateBadgeSVG(tt.pct)
		if !strings.Contains(svg, tt.wantColor) {
			t.Errorf("SVG for %.1f%% should contain color %q", tt.pct, tt.wantColor)
		}
	}
}

func TestGenerateBadgeSVGStructure(t *testing.T) {
	svg := GenerateBadgeSVG(50.0)

	// Verify essential SVG elements
	checks := []string{
		`xmlns="http://www.w3.org/2000/svg"`,
		`role="img"`,
		`<title>`,
		`<linearGradient`,
		`<clipPath`,
		`font-family="Verdana`,
	}
	for _, check := range checks {
		if !strings.Contains(svg, check) {
			t.Errorf("SVG should contain %q", check)
		}
	}
}
