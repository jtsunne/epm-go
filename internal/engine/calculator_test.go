package engine

import (
	"testing"
	"time"

	"github.com/dm/epm-go/internal/client"
	"github.com/dm/epm-go/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestClampRate(t *testing.T) {
	cases := []struct {
		name  string
		input float64
		want  float64
	}{
		{"zero", 0, 0},
		{"normal", 1000, 1000},
		{"at limit", maxRatePerSec, maxRatePerSec},
		{"above limit", maxRatePerSec + 1, 0},
		{"huge value", 1e12, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, clampRate(tc.input))
		})
	}
}

func TestClampLatency(t *testing.T) {
	cases := []struct {
		name  string
		input float64
		want  float64
	}{
		{"zero", 0, 0},
		{"normal", 5.5, 5.5},
		{"at limit", maxLatencyMs, maxLatencyMs},
		{"above limit", maxLatencyMs + 1, maxLatencyMs},
		{"huge value", 1e9, maxLatencyMs},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, clampLatency(tc.input))
		})
	}
}

func TestSafeDivide(t *testing.T) {
	cases := []struct {
		name string
		a, b float64
		want float64
	}{
		{"normal", 10, 4, 2.5},
		{"divide by zero", 5, 0, 0},
		{"zero numerator", 0, 5, 0},
		{"both zero", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, safeDivide(tc.a, tc.b))
		})
	}
}

func TestMaxFloat64(t *testing.T) {
	cases := []struct {
		name string
		a, b float64
		want float64
	}{
		{"a greater", 5, 3, 5},
		{"b greater", 3, 5, 5},
		{"equal", 4, 4, 4},
		{"negative", -1, -2, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, maxFloat64(tc.a, tc.b))
		})
	}
}

// makeNodeStats builds a NodeStatsResponse with a single node for testing.
func makeNodeStats(indexOps, indexTimeMs, searchOps, searchTimeMs int64) client.NodeStatsResponse {
	return client.NodeStatsResponse{
		Nodes: map[string]client.NodePerformanceStats{
			"node1": {
				Name: "node1",
				Indices: &client.NodeIndicesStats{
					Indexing: client.NodeIndexingStats{
						IndexTotal:        indexOps,
						IndexTimeInMillis: indexTimeMs,
					},
					Search: client.NodeSearchStats{
						QueryTotal:        searchOps,
						QueryTimeInMillis: searchTimeMs,
					},
				},
			},
		},
	}
}

func TestCalcClusterMetrics_FirstSnapshot(t *testing.T) {
	curr := &model.Snapshot{
		NodeStats: makeNodeStats(1000, 500, 2000, 800),
		FetchedAt: time.Now(),
	}
	got := CalcClusterMetrics(nil, curr, 10*time.Second)
	assert.Equal(t, model.PerformanceMetrics{}, got)
}

func TestCalcClusterMetrics_BasicRates(t *testing.T) {
	// prev: 1000 index ops, 500ms; 2000 search ops, 800ms
	// curr: 2000 index ops, 700ms; 3500 search ops, 1300ms
	// elapsed: 10s
	// indexingRate = (2000-1000)/10 = 100 /s
	// searchRate   = (3500-2000)/10 = 150 /s
	// indexLatency = (700-500)/(2000-1000) = 200/1000 = 0.2 ms
	// searchLatency = (1300-800)/(3500-2000) = 500/1500 ≈ 0.333 ms
	prev := &model.Snapshot{
		NodeStats: makeNodeStats(1000, 500, 2000, 800),
	}
	curr := &model.Snapshot{
		NodeStats: makeNodeStats(2000, 700, 3500, 1300),
	}
	elapsed := 10 * time.Second
	got := CalcClusterMetrics(prev, curr, elapsed)

	assert.InDelta(t, 100.0, got.IndexingRate, 1e-9)
	assert.InDelta(t, 150.0, got.SearchRate, 1e-9)
	assert.InDelta(t, 0.2, got.IndexLatency, 1e-9)
	assert.InDelta(t, 500.0/1500.0, got.SearchLatency, 1e-9)
}

func TestCalcClusterMetrics_CounterReset(t *testing.T) {
	// curr ops < prev ops → delta is negative → clamped to 0 → rate = 0
	prev := &model.Snapshot{
		NodeStats: makeNodeStats(5000, 2000, 8000, 3000),
	}
	curr := &model.Snapshot{
		NodeStats: makeNodeStats(100, 50, 200, 80),
	}
	got := CalcClusterMetrics(prev, curr, 10*time.Second)
	assert.Equal(t, 0.0, got.IndexingRate)
	assert.Equal(t, 0.0, got.SearchRate)
}

func TestCalcClusterMetrics_TooShortInterval(t *testing.T) {
	prev := &model.Snapshot{
		NodeStats: makeNodeStats(1000, 500, 2000, 800),
	}
	curr := &model.Snapshot{
		NodeStats: makeNodeStats(2000, 700, 3500, 1300),
	}
	// 500ms < 1s → return zeros
	got := CalcClusterMetrics(prev, curr, 500*time.Millisecond)
	assert.Equal(t, model.PerformanceMetrics{}, got)
}

func TestCalcClusterMetrics_RateSanityCap(t *testing.T) {
	// delta = maxRatePerSec*10+1 ops in 1 second → rate exceeds cap → clamped to 0
	bigDelta := int64(maxRatePerSec*10 + 1)
	prev := &model.Snapshot{
		NodeStats: makeNodeStats(0, 0, 0, 0),
	}
	curr := &model.Snapshot{
		NodeStats: makeNodeStats(bigDelta, 0, bigDelta, 0),
	}
	got := CalcClusterMetrics(prev, curr, 1*time.Second)
	assert.Equal(t, 0.0, got.IndexingRate)
	assert.Equal(t, 0.0, got.SearchRate)
}

func TestCalcClusterMetrics_LatencySanityCap(t *testing.T) {
	// 1 op with enormous time → raw latency >> maxLatencyMs → capped
	hugeTime := int64(maxLatencyMs*2 + 1)
	prev := &model.Snapshot{
		NodeStats: makeNodeStats(0, 0, 0, 0),
	}
	curr := &model.Snapshot{
		NodeStats: makeNodeStats(1, hugeTime, 1, hugeTime),
	}
	got := CalcClusterMetrics(prev, curr, 10*time.Second)
	assert.Equal(t, maxLatencyMs, got.IndexLatency)
	assert.Equal(t, maxLatencyMs, got.SearchLatency)
}
