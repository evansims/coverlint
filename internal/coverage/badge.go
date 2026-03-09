package coverage

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// clampPct normalizes a coverage percentage to a safe value.
func clampPct(pct float64) float64 {
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		return 0
	}
	return pct
}

// badgeColor returns a color based on coverage percentage,
// using the conventional thresholds for coverage badges.
func badgeColor(pct float64) string {
	switch {
	case pct >= 90:
		return "brightgreen"
	case pct >= 80:
		return "green"
	case pct >= 70:
		return "yellowgreen"
	case pct >= 60:
		return "yellow"
	case pct >= 50:
		return "orange"
	default:
		return "red"
	}
}

// badgeColorHex returns the hex color for SVG rendering.
func badgeColorHex(pct float64) string {
	switch {
	case pct >= 90:
		return "#4c1"
	case pct >= 80:
		return "#97ca00"
	case pct >= 70:
		return "#a4a61d"
	case pct >= 60:
		return "#dfb317"
	case pct >= 50:
		return "#fe7d37"
	default:
		return "#e05d44"
	}
}

// shieldsEndpoint is the JSON format for shields.io endpoint badges.
type shieldsEndpoint struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

// GenerateBadgeJSON returns a shields.io endpoint-compatible JSON string
// for the given coverage percentage.
func GenerateBadgeJSON(pct float64) string {
	pct = clampPct(pct)
	endpoint := shieldsEndpoint{
		SchemaVersion: 1,
		Label:         "coverage",
		Message:       fmt.Sprintf("%.0f%%", pct),
		Color:         badgeColor(pct),
	}
	data, _ := json.Marshal(endpoint)
	return string(data)
}

// GenerateBadgeSVG returns an SVG badge string for the given coverage percentage.
// All interpolated values are derived from controlled sources (numeric formatting
// and hardcoded strings) — no user-controlled text reaches the SVG template.
func GenerateBadgeSVG(pct float64) string {
	pct = clampPct(pct)
	label := "coverage"
	value := fmt.Sprintf("%.0f%%", pct)
	color := badgeColorHex(pct)

	// Approximate text widths using Verdana 11px metrics.
	// Average character width ~6.8px, with padding.
	labelWidth := len(label)*7 + 10
	valueWidth := len(value)*7 + 10
	totalWidth := labelWidth + valueWidth

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="%s: %s">`, totalWidth, label, value)
	sb.WriteString(`<title>`)
	fmt.Fprintf(&sb, "%s: %s", label, value)
	sb.WriteString(`</title>`)
	sb.WriteString(`<linearGradient id="s" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`)
	sb.WriteString(`<clipPath id="r"><rect width="`)
	fmt.Fprintf(&sb, `%d`, totalWidth)
	sb.WriteString(`" height="20" rx="3" fill="#fff"/></clipPath>`)
	sb.WriteString(`<g clip-path="url(#r)">`)
	fmt.Fprintf(&sb, `<rect width="%d" height="20" fill="#555"/>`, labelWidth)
	fmt.Fprintf(&sb, `<rect x="%d" width="%d" height="20" fill="%s"/>`, labelWidth, valueWidth, color)
	fmt.Fprintf(&sb, `<rect width="%d" height="20" fill="url(#s)"/>`, totalWidth)
	sb.WriteString(`</g>`)
	sb.WriteString(`<g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" text-rendering="geometricPrecision" font-size="110">`)
	fmt.Fprintf(&sb, `<text aria-hidden="true" x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)">%s</text>`, (labelWidth*10)/2, label)
	fmt.Fprintf(&sb, `<text x="%d" y="140" transform="scale(.1)">%s</text>`, (labelWidth*10)/2, label)
	fmt.Fprintf(&sb, `<text aria-hidden="true" x="%d" y="150" fill="#010101" fill-opacity=".3" transform="scale(.1)">%s</text>`, labelWidth*10+(valueWidth*10)/2, value)
	fmt.Fprintf(&sb, `<text x="%d" y="140" transform="scale(.1)">%s</text>`, labelWidth*10+(valueWidth*10)/2, value)
	sb.WriteString(`</g></svg>`)

	return sb.String()
}
