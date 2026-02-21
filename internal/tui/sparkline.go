package tui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// sparkBlocks is the 8-level braille block character set for sparklines.
var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// RenderSparkline converts a slice of float64 values into a braille-block sparkline
// string of exactly `width` characters. The sparkline is colored using the provided
// lipgloss color style.
//
// Rules:
//   - Empty values → return width spaces
//   - All zeros → return all '▁' (floor level)
//   - Values longer than width → use last width values
//   - Fewer values than width → left-pad with spaces
func RenderSparkline(values []float64, width int, color lipgloss.Color) string {
	if width <= 0 {
		return ""
	}

	if len(values) == 0 {
		return strings.Repeat(" ", width)
	}

	// Take last `width` values if the slice is longer.
	if len(values) > width {
		values = values[len(values)-width:]
	}

	maxVal := slices.Max(values)

	style := lipgloss.NewStyle().Foreground(color)

	var sb strings.Builder
	// Left-pad with spaces when fewer values than width.
	padLen := width - len(values)
	sb.WriteString(strings.Repeat(" ", padLen))

	for _, v := range values {
		var idx int
		if maxVal > 0 {
			idx = int(v / maxVal * 7)
		}
		// Clamp to [0, 7].
		if idx < 0 {
			idx = 0
		}
		if idx > 7 {
			idx = 7
		}
		sb.WriteRune(sparkBlocks[idx])
	}

	return style.Render(sb.String())
}
