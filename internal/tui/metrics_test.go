package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dm/epm-go/internal/model"
)

func TestRenderMetricCard_ContainsTitle(t *testing.T) {
	title := "Indexing Rate"
	result := renderMetricCard(title, "1,234.5", "/s", []float64{1, 2, 3}, 30, colorGreen)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, title)
}

func TestRenderMetricCard_ContainsValue(t *testing.T) {
	value := "987.6"
	result := renderMetricCard("Search Rate", value, "/s", []float64{5, 4, 3}, 30, colorCyan)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, value)
}

func TestRenderMetricCard_ContainsUnit(t *testing.T) {
	result := renderMetricCard("Index Latency", "3.21", "ms", []float64{1, 2}, 30, colorYellow)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "ms")
}

func TestRenderMetricCard_NoUnit(t *testing.T) {
	// When unit is empty the value should still appear without trailing space issues.
	result := renderMetricCard("Search Latency", "5.00", "", nil, 30, colorRed)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "5.00")
}

func TestRenderMetricCard_MinWidthEnforced(t *testing.T) {
	// Card width below 20 should still render without panicking.
	result := renderMetricCard("Rate", "1.0", "", nil, 5, colorGreen)
	require.NotEmpty(t, result)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "Rate")
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
}

func TestRenderMetricsRow_NarrowTerminal(t *testing.T) {
	// width < 80 triggers 2x2 grid layout — should not panic and should contain all titles.
	app := NewApp(nil, 10*time.Second)
	app.width = 60

	snap := &model.Snapshot{FetchedAt: time.Now()}
	app.current = snap
	app.metrics = model.PerformanceMetrics{}

	result := renderMetricsRow(app)
	require.NotEmpty(t, result)

	stripped := stripANSI(result)
	assert.Contains(t, stripped, "Indexing Rate")
	assert.Contains(t, stripped, "Search Latency")
}
