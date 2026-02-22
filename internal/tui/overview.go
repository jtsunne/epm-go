package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/dm/epm-go/internal/format"
)

// maxCardHeight is the tallest card content in wide mode.
// Card 7 (Storage) has 4 lines: value + bar + used/total + label.
// All other cards are padded to this height for visual consistency.
const maxCardHeight = 4

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
	if width < 6 {
		// Paired cards need at least Width(3) each (inner=1) so lipgloss can
		// hard-wrap single-word content without overflowing. Below 6, the inner
		// content space is 0 or negative and unbreakable strings (bar chars,
		// metric values) overflow the terminal width.
		return ""
	}

	narrowMode := width < 80

	// applyHeight sets a fixed height + vertical centering in wide mode so that
	// all 7 cards share the same number of lines and lipgloss fills the card
	// background evenly. Narrow mode skips this — JoinHorizontal per row already
	// equalises heights within each pair.
	applyHeight := func(s lipgloss.Style) lipgloss.Style {
		if !narrowMode {
			return s.Height(maxCardHeight).AlignVertical(lipgloss.Center)
		}
		return s
	}

	// Build per-card width slice (7 elements, indices 0-6).
	// Width() in lipgloss sets the outer rendered width (including padding),
	// so 7 cards each with Width(w) sum to exactly sum(w).
	cardWidths := make([]int, 7)
	if narrowMode {
		// 2 cards per row; remainder (+1) goes to first card of each pair.
		// Card 7 (index 6) spans the full row alone.
		narrowBase := width / 2
		narrowRem := width % 2
		// Preferred minimum of 4 per paired card, but cap at narrowBase so the
		// pair (2 * cardWidth) never exceeds the terminal width. At ultra-narrow
		// widths (narrowBase < 4) use narrowBase directly; floor at 1 to prevent
		// zero-width cards.
		minPaired := 4
		if narrowBase < minPaired {
			minPaired = narrowBase
		}
		if minPaired < 1 {
			minPaired = 1
		}
		for i := range cardWidths {
			switch {
			case i == 6:
				cardWidths[i] = width
			case i%2 == 0:
				cardWidths[i] = narrowBase + narrowRem
			default:
				cardWidths[i] = narrowBase
			}
			// Enforce minimum card width for paired cards without overflowing terminal.
			if i < 6 && cardWidths[i] < minPaired {
				cardWidths[i] = minPaired
			}
		}
	} else {
		// Wide mode: distribute width evenly across 7 cards.
		// First (width % 7) cards each get +1 to absorb the remainder.
		base := width / 7
		rem := width % 7
		for i := range cardWidths {
			if i < rem {
				cardWidths[i] = base + 1
			} else {
				cardWidths[i] = base
			}
			if cardWidths[i] < 8 {
				cardWidths[i] = 8
			}
		}
	}

	// barWidthFor returns the mini-bar inner width for card at index i.
	barWidthFor := func(i int) int {
		bw := cardWidths[i] - 4
		if bw < 4 {
			bw = 4
		}
		return bw
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
	card1 := applyHeight(StyleOverviewCard.
		Background(statusBg).
		Foreground(colorDark).
		Bold(true).
		Width(cardWidths[0])).
		Render(statusText + "\nStatus")

	// Card 2: Node count — blue foreground.
	card2 := applyHeight(StyleOverviewCard.
		Foreground(colorBlue).
		Width(cardWidths[1])).
		Render(fmt.Sprintf("%d", health.NumberOfNodes) + "\nNodes")

	// Card 3: Index count — purple foreground.
	indexCount := len(app.current.Indices)
	card3 := applyHeight(StyleOverviewCard.
		Foreground(colorPurple).
		Width(cardWidths[2])).
		Render(fmt.Sprintf("%d", indexCount) + "\nIndices")

	// Card 4: Active shards — indigo foreground.
	card4 := applyHeight(StyleOverviewCard.
		Foreground(colorIndigo).
		Width(cardWidths[3])).
		Render(fmt.Sprintf("%d", health.ActiveShards) + "\nActive Shards")

	// Card 5: CPU% with mini bar — threshold-colored via cpuSeverity.
	cpuPct := res.AvgCPUPercent
	cpuSev := cpuSeverity(cpuPct)
	cpuVal := fmt.Sprintf("%.1f%%", cpuPct)
	if cpuSev == severityCritical {
		cpuVal += "!"
	}
	cpuBar := renderMiniBar(cpuPct, barWidthFor(4))
	card5 := applyHeight(severityCardStyle().
		Foreground(severityFg(cpuSev)).
		Width(cardWidths[4])).
		Render(cpuVal + "\n" + cpuBar + "\nCPU")

	// Card 6: JVM heap% with mini bar — threshold-colored via jvmSeverity.
	jvmPct := res.AvgJVMHeapPercent
	jvmSev := jvmSeverity(jvmPct)
	jvmVal := fmt.Sprintf("%.1f%%", jvmPct)
	if jvmSev == severityCritical {
		jvmVal += "!"
	}
	jvmBar := renderMiniBar(jvmPct, barWidthFor(5))
	card6 := applyHeight(severityCardStyle().
		Foreground(severityFg(jvmSev)).
		Width(cardWidths[5])).
		Render(jvmVal + "\n" + jvmBar + "\nJVM Heap")

	// Card 7: Storage% with mini bar — threshold-colored via storageSeverity.
	storagePct := res.StoragePercent
	storageSev := storageSeverity(storagePct)
	storageVal := fmt.Sprintf("%.1f%%", storagePct)
	if storageSev == severityCritical {
		storageVal += "!"
	}
	storageBar := renderMiniBar(storagePct, barWidthFor(6))
	usedStr := format.FormatBytes(res.StorageUsedBytes)
	totalStr := format.FormatBytes(res.StorageTotalBytes)
	card7 := applyHeight(severityCardStyle().
		Foreground(severityFg(storageSev)).
		Width(cardWidths[6])).
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
