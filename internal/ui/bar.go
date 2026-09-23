package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorGreen  = lipgloss.Color("#00E676")
	colorYellow = lipgloss.Color("#FFD600")
	colorOrange = lipgloss.Color("#FF9100")
	colorRed    = lipgloss.Color("#FF5252")
)

func RenderBar(ratio float64, width int, colored bool) string {
	if width <= 0 {
		return ""
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	totalSteps := float64(width) * ratio
	fullBlocks := int(totalSteps)
	remainder := totalSteps - float64(fullBlocks)

	var sb strings.Builder
	sb.WriteString(strings.Repeat("█", fullBlocks))

	if fullBlocks < width {
		if remainder >= 0.5 {
			sb.WriteString("▌")
			fullBlocks++
		}
		if width-fullBlocks > 0 {
			sb.WriteString(strings.Repeat("░", width-fullBlocks))
		}
	}

	barStr := sb.String()
	if !colored {
		return barStr
	}

	var style lipgloss.Style
	switch {
	case ratio >= 0.50:
		style = lipgloss.NewStyle().Foreground(colorRed)
	case ratio >= 0.25:
		style = lipgloss.NewStyle().Foreground(colorOrange)
	case ratio >= 0.10:
		style = lipgloss.NewStyle().Foreground(colorYellow)
	default:
		style = lipgloss.NewStyle().Foreground(colorGreen)
	}

	return style.Render(barStr)
}

func FormatPercent(ratio float64) string {
	pct := ratio * 100
	return fmt.Sprintf("%5.1f%%", pct)
}
