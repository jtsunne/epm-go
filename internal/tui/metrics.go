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

	// Inner width = card width minus border (2) and padding (2).
	// lipgloss Width() includes padding in its measurement, so available content
	// width = Width - padding = (cardWidth-4) - 2 = cardWidth-6.
	innerWidth := cardWidth - 6
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
		Width(cardWidth - 4)

	return cardStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		valueLine,
		sparkLine,
	))
}

// renderMetricsRow renders 4 metric cards (Indexing Rate, Search Rate,
// Index Latency, Search Latency) with a "Cluster Performance" section label.
// Wide terminals (>= 80 cols): 1x4 horizontal row.
// Narrow terminals (< 80 cols): 2x2 grid.
// Returns empty string when no data is available.
func renderMetricsRow(app *App) string {
	if app.current == nil {
		return ""
	}

	label := StyleDim.Render("Cluster Performance")

	if app.width < 80 {
		// 2x2 grid layout for narrow terminals.
		// Each card renders at (cardWidth-2) chars wide (lipgloss Width includes padding,
		// border adds 2). For 2 cards to fill app.width: 2*(cardWidth-2)=app.width → cardWidth=(app.width+4)/2.
		cardWidth := (app.width + 4) / 2
		if cardWidth < 20 {
			cardWidth = 20
		}
		top := lipgloss.JoinHorizontal(lipgloss.Top,
			renderMetricCard("Indexing Rate", format.FormatRate(app.metrics.IndexingRate), "", app.history.Values("indexingRate"), cardWidth, colorGreen),
			renderMetricCard("Search Rate", format.FormatRate(app.metrics.SearchRate), "", app.history.Values("searchRate"), cardWidth, colorCyan),
		)
		bottom := lipgloss.JoinHorizontal(lipgloss.Top,
			renderMetricCard("Index Latency", format.FormatLatency(app.metrics.IndexLatency), "", app.history.Values("indexLatency"), cardWidth, colorYellow),
			renderMetricCard("Search Latency", format.FormatLatency(app.metrics.SearchLatency), "", app.history.Values("searchLatency"), cardWidth, colorRed),
		)
		return lipgloss.JoinVertical(lipgloss.Left, label, top, bottom)
	}

	// 1x4 horizontal row for wide terminals.
	// Each card renders at (cardWidth-2) chars wide (lipgloss Width includes padding,
	// border adds 2). For 4 cards to fill app.width: 4*(cardWidth-2)=app.width → cardWidth=(app.width+8)/4.
	cardWidth := (app.width + 8) / 4
	if cardWidth < 20 {
		cardWidth = 20
	}

	cards := []string{
		renderMetricCard("Indexing Rate", format.FormatRate(app.metrics.IndexingRate), "", app.history.Values("indexingRate"), cardWidth, colorGreen),
		renderMetricCard("Search Rate", format.FormatRate(app.metrics.SearchRate), "", app.history.Values("searchRate"), cardWidth, colorCyan),
		renderMetricCard("Index Latency", format.FormatLatency(app.metrics.IndexLatency), "", app.history.Values("indexLatency"), cardWidth, colorYellow),
		renderMetricCard("Search Latency", format.FormatLatency(app.metrics.SearchLatency), "", app.history.Values("searchLatency"), cardWidth, colorRed),
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, cards...)
	return lipgloss.JoinVertical(lipgloss.Left, label, row)
}
