package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtsunne/epm-go/internal/model"
)

func TestRenderMetricCard_ContainsTitle(t *testing.T) {
	title := "Indexing Rate"
	result := renderMetricCard(title, "1,234.5", "/s", []float64{1, 2, 3}, 30, colorGreen, StyleDim)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, title)
}

func TestRenderMetricCard_ContainsValue(t *testing.T) {
	value := "987.6"
	result := renderMetricCard("Search Rate", value, "/s", []float64{5, 4, 3}, 30, colorCyan, StyleDim)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, value)
}

func TestRenderMetricCard_ContainsUnit(t *testing.T) {
	result := renderMetricCard("Index Latency", "3.21", "ms", []float64{1, 2}, 30, colorYellow, StyleDim)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "ms")
}

func TestRenderMetricCard_NoUnit(t *testing.T) {
	// When unit is empty the value should still appear without trailing space issues.
	result := renderMetricCard("Search Latency", "5.00", "", nil, 30, colorRed, StyleDim)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "5.00")
}

func TestRenderMetricCard_MinWidthEnforced(t *testing.T) {
	// Card width below the internal minimum should still render without panicking.
	// Content may wrap at very narrow widths — only check non-empty output.
	result := renderMetricCard("Rate", "1.0", "", nil, 5, colorGreen, StyleDim)
	require.NotEmpty(t, result)
}

func TestRenderMetricsRow_NilSnapshot(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120
	// current is nil — must return empty string
	assert.Equal(t, "", renderMetricsRow(app))
}

func TestRenderMetricsRow_WithSnapshot(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120

	snap := &model.Snapshot{FetchedAt: time.Now()}
	app.current = snap
	app.metrics = model.PerformanceMetrics{
		IndexingRate:  1500,
		SearchRate:    800,
		IndexLatency:  2.5,
		SearchLatency: 7.3,
	}

	result := renderMetricsRow(app)
	require.NotEmpty(t, result)

	stripped := stripANSI(result)
	assert.Contains(t, stripped, "Indexing Rate")
	assert.Contains(t, stripped, "Search Rate")
	assert.Contains(t, stripped, "Index Latency")
	assert.Contains(t, stripped, "Search Latency")
	assert.Contains(t, stripped, "Cluster Performance")
	// Verify formatted metric values appear in the output.
	assert.Contains(t, stripped, "1,500.0")
	assert.Contains(t, stripped, "800.0")
	assert.Contains(t, stripped, "2.50")
	assert.Contains(t, stripped, "7.30")
}

func TestRenderMetricsRow_NarrowTerminal(t *testing.T) {
	// width < 80 triggers 2x2 grid layout — should not panic or overflow.
	// Widths >= 20 guarantee "Rate" (4 chars) fits on one card line without wrapping.
	narrowWidths := []int{60, 40, 30, 20}
	for _, w := range narrowWidths {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			app := NewApp(nil, 10*time.Second)
			app.width = w

			snap := &model.Snapshot{FetchedAt: time.Now()}
			app.current = snap
			app.metrics = model.PerformanceMetrics{}

			result := renderMetricsRow(app)
			require.NotEmpty(t, result)

			stripped := stripANSI(result)
			// Section label and "Rate" (4 chars) fit on one line at all tested widths.
			assert.Contains(t, stripped, "Cluster Performance")
			assert.Contains(t, stripped, "Rate")

			// No line in the rendered output may exceed app.width columns.
			for i, line := range strings.Split(result, "\n") {
				got := lipgloss.Width(line)
				assert.LessOrEqual(t, got, w, "narrow metrics line %d width %d > terminal %d", i, got, w)
			}
		})
	}
}

func TestRenderMetricsRow_NarrowEdgeWidths(t *testing.T) {
	// width=12 is the minimum that renders the 2x2 grid (cardWidth=8, 2*(8-2)=12).
	// Content wraps severely at this width so only the no-overflow invariant is checked.
	edgeWidths := []int{12, 13, 14, 15}
	for _, w := range edgeWidths {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			app := NewApp(nil, 10*time.Second)
			app.width = w

			snap := &model.Snapshot{FetchedAt: time.Now()}
			app.current = snap
			app.metrics = model.PerformanceMetrics{}

			result := renderMetricsRow(app)
			require.NotEmpty(t, result)

			// No line in the rendered output may exceed app.width columns.
			for i, line := range strings.Split(result, "\n") {
				got := lipgloss.Width(line)
				assert.LessOrEqual(t, got, w, "edge metrics line %d width %d > terminal %d", i, got, w)
			}
		})
	}
}

func TestRenderMetricsRow_TooNarrowReturnsEmpty(t *testing.T) {
	// widths < 12 cannot fit two 8-wide cards without overflow — must return empty string.
	tooNarrowWidths := []int{11, 10, 8, 4, 1}
	for _, w := range tooNarrowWidths {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			app := NewApp(nil, 10*time.Second)
			app.width = w

			snap := &model.Snapshot{FetchedAt: time.Now()}
			app.current = snap
			app.metrics = model.PerformanceMetrics{}

			result := renderMetricsRow(app)
			assert.Equal(t, "", result, "expected empty string for too-narrow terminal width=%d", w)
		})
	}
}
