package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dm/epm-go/internal/format"
)

// renderOverview renders the 7-stat horizontal overview bar.
// Returns empty string if no snapshot is available yet.
func renderOverview(app *App) string {
	if app.current == nil {
		return ""
	}

	width := app.width
	if width <= 0 {
		width = 80
	}

	cardWidth := (width - 14) / 7
	if cardWidth < 8 {
		cardWidth = 8
	}

	// Mini bar inner width: card width minus padding (1 char each side).
	barWidth := cardWidth - 4
	if barWidth < 4 {
		barWidth = 4
	}

	health := app.current.Health
	res := app.resources

	// Card 1: Cluster Status — colored background.
	statusText := strings.ToUpper(health.Status)
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

	// Card 5: CPU% with mini bar — yellow >80%, red >90%.
	cpuPct := res.AvgCPUPercent
	cpuFg := colorWhite
	if cpuPct > 90 {
		cpuFg = colorRed
	} else if cpuPct > 80 {
		cpuFg = colorYellow
	}
	cpuBar := renderMiniBar(cpuPct, barWidth)
	card5 := StyleOverviewCard.
		Foreground(cpuFg).
		Width(cardWidth).
		Render(fmt.Sprintf("%.1f%%", cpuPct) + "\n" + cpuBar + "\nCPU")

	// Card 6: JVM heap% with mini bar — yellow >75%, red >85%.
	jvmPct := res.AvgJVMHeapPercent
	jvmFg := colorWhite
	if jvmPct > 85 {
		jvmFg = colorRed
	} else if jvmPct > 75 {
		jvmFg = colorYellow
	}
	jvmBar := renderMiniBar(jvmPct, barWidth)
	card6 := StyleOverviewCard.
		Foreground(jvmFg).
		Width(cardWidth).
		Render(fmt.Sprintf("%.1f%%", jvmPct) + "\n" + jvmBar + "\nJVM Heap")

	// Card 7: Storage% with mini bar — yellow >80%, red >90%.
	storagePct := res.StoragePercent
	storageFg := colorWhite
	if storagePct > 90 {
		storageFg = colorRed
	} else if storagePct > 80 {
		storageFg = colorYellow
	}
	storageBar := renderMiniBar(storagePct, barWidth)
	usedStr := format.FormatBytes(res.StorageUsedBytes)
	totalStr := format.FormatBytes(res.StorageTotalBytes)
	card7 := StyleOverviewCard.
		Foreground(storageFg).
		Width(cardWidth).
		Render(fmt.Sprintf("%.1f%%", storagePct) + "\n" + storageBar + "\n" + usedStr + "/" + totalStr)

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
