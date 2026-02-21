package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// renderMetricCard renders a single metric card with title, value, and sparkline.
//
// Layout (3 rows inside a rounded border):
//
//	╭──────────────────╮
//	│ Title            │   ← dim/muted color
//	│ 1,204.3 /s       │   ← bold, metric color
//	│ ▁▂▃▅▇█▇▅▃▂       │   ← colored sparkline
//	╰──────────────────╯
func renderMetricCard(title, value, unit string, sparkValues []float64, cardWidth int, color lipgloss.Color) string {
	const minCardWidth = 20
	if cardWidth < minCardWidth {
		cardWidth = minCardWidth
	}

	// Inner width = card width minus border (2) and padding (2 × 1 side = 2).
	innerWidth := cardWidth - 4
	if innerWidth < 1 {
		innerWidth = 1
	}

	valueStyle := lipgloss.NewStyle().Bold(true).Foreground(color)

	titleLine := StyleDim.Render(title)

	var valueLine string
	if unit != "" {
		valueLine = valueStyle.Render(value + " " + unit)
	} else {
		valueLine = valueStyle.Render(value)
	}

	sparkLine := RenderSparkline(sparkValues, innerWidth, color)

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorGray).
		Padding(0, 1).
		Width(cardWidth)

	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		valueLine,
		sparkLine,
	))
}
