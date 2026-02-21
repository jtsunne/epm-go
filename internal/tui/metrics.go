package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/dm/epm-go/internal/format"
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

// renderMetricsRow renders 4 metric cards (Indexing Rate, Search Rate,
// Index Latency, Search Latency) horizontally with a "Cluster Performance"
// section label above them. Returns empty string when no data is available.
func renderMetricsRow(app *App) string {
	if app.current == nil {
		return ""
	}

	cardWidth := (app.width - 8) / 4
	if cardWidth < 20 {
		cardWidth = 20
	}

	cards := []string{
		renderMetricCard(
			"Indexing Rate",
			format.FormatRate(app.metrics.IndexingRate),
			"",
			app.history.Values("indexingRate"),
			cardWidth,
			colorGreen,
		),
		renderMetricCard(
			"Search Rate",
			format.FormatRate(app.metrics.SearchRate),
			"",
			app.history.Values("searchRate"),
			cardWidth,
			colorCyan,
		),
		renderMetricCard(
			"Index Latency",
			format.FormatLatency(app.metrics.IndexLatency),
			"",
			app.history.Values("indexLatency"),
			cardWidth,
			colorYellow,
		),
		renderMetricCard(
			"Search Latency",
			format.FormatLatency(app.metrics.SearchLatency),
			"",
			app.history.Values("searchLatency"),
			cardWidth,
			colorRed,
		),
	}

	label := StyleDim.Render("Cluster Performance")
	row := lipgloss.JoinHorizontal(lipgloss.Top, cards...)
	return lipgloss.JoinVertical(lipgloss.Left, label, row)
}
