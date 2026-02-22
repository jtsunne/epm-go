package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dm/epm-go/internal/format"
)

// renderOverview renders the 7-stat overview bar.
// Wide terminals (>= 80 cols): all 7 cards in a single horizontal row.
// Narrow terminals (< 80 cols): cards stacked in rows of 2 (4 rows: 2+2+2+1).
// Returns empty string if no snapshot is available yet.
func renderOverview(app *App) string {
	if app.current == nil {
		return ""
	}

	width := app.width
	if width <= 0 {
		width = 80
	}

	narrowMode := width < 80

	var cardWidth int
	if narrowMode {
		// 2 cards per row: split width evenly between 2 cards.
		cardWidth = (width - 4) / 2
		if cardWidth < 10 {
			cardWidth = 10
		}
	} else {
		cardWidth = (width - 14) / 7
		if cardWidth < 8 {
			cardWidth = 8
		}
	}

	// Mini bar inner width: card width minus padding (1 char each side).
	barWidth := cardWidth - 4
	if barWidth < 4 {
		barWidth = 4
	}

	health := app.current.Health
	res := app.resources

	// Card 1: Cluster Status — colored background.
	statusText := strings.ToUpper(sanitize(health.Status))
	if statusText == "" {
		statusText = "UNKNOWN"
	}
	var statusBg lipgloss.Color
	switch health.Status {
	case "green":
		statusBg = colorGreen
	case "yellow":
		statusBg = colorYellow
	case "red":
		statusBg = colorRed
	default:
		statusBg = colorGray
	}
	card1 := StyleOverviewCard.
		Background(statusBg).
		Foreground(colorDark).
		Bold(true).
		Width(cardWidth).
		Render(statusText + "\nStatus")

	// Card 2: Node count — blue foreground.
	card2 := StyleOverviewCard.
		Foreground(colorBlue).
		Width(cardWidth).
		Render(fmt.Sprintf("%d", health.NumberOfNodes) + "\nNodes")

	// Card 3: Index count — purple foreground.
	indexCount := len(app.current.Indices)
	card3 := StyleOverviewCard.
		Foreground(colorPurple).
		Width(cardWidth).
		Render(fmt.Sprintf("%d", indexCount) + "\nIndices")

	// Card 4: Active shards — indigo foreground.
	card4 := StyleOverviewCard.
		Foreground(colorIndigo).
		Width(cardWidth).
		Render(fmt.Sprintf("%d", health.ActiveShards) + "\nActive Shards")

	// Card 5: CPU% with mini bar — threshold-colored via cpuSeverity.
	cpuPct := res.AvgCPUPercent
	cpuSev := cpuSeverity(cpuPct)
	cpuVal := fmt.Sprintf("%.1f%%", cpuPct)
	if cpuSev == severityCritical {
		cpuVal += "!"
	}
	cpuBar := renderMiniBar(cpuPct, barWidth)
	card5 := severityCardStyle().
		Foreground(severityFg(cpuSev)).
		Width(cardWidth).
		Render(cpuVal + "\n" + cpuBar + "\nCPU")

	// Card 6: JVM heap% with mini bar — threshold-colored via jvmSeverity.
	jvmPct := res.AvgJVMHeapPercent
	jvmSev := jvmSeverity(jvmPct)
	jvmVal := fmt.Sprintf("%.1f%%", jvmPct)
	if jvmSev == severityCritical {
		jvmVal += "!"
	}
	jvmBar := renderMiniBar(jvmPct, barWidth)
	card6 := severityCardStyle().
		Foreground(severityFg(jvmSev)).
		Width(cardWidth).
		Render(jvmVal + "\n" + jvmBar + "\nJVM Heap")

	// Card 7: Storage% with mini bar — threshold-colored via storageSeverity.
	storagePct := res.StoragePercent
	storageSev := storageSeverity(storagePct)
	storageVal := fmt.Sprintf("%.1f%%", storagePct)
	if storageSev == severityCritical {
		storageVal += "!"
	}
	storageBar := renderMiniBar(storagePct, barWidth)
	usedStr := format.FormatBytes(res.StorageUsedBytes)
	totalStr := format.FormatBytes(res.StorageTotalBytes)
	card7 := severityCardStyle().
		Foreground(severityFg(storageSev)).
		Width(cardWidth).
		Render(storageVal + "\n" + storageBar + "\n" + usedStr + "/" + totalStr + "\nStorage")

	if narrowMode {
		// Arrange 7 cards in rows of 2 (4 rows: 2+2+2+1).
		row1 := lipgloss.JoinHorizontal(lipgloss.Top, card1, card2)
		row2 := lipgloss.JoinHorizontal(lipgloss.Top, card3, card4)
		row3 := lipgloss.JoinHorizontal(lipgloss.Top, card5, card6)
		return lipgloss.JoinVertical(lipgloss.Left, row1, row2, row3, card7)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, card1, card2, card3, card4, card5, card6, card7)
}

// renderMiniBar renders a mini progress bar using Unicode block characters.
// Fills proportionally using "█" (U+2588) for filled and "░" (U+2591) for empty cells.
func renderMiniBar(percent float64, width int) string {
	if width <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(percent / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
