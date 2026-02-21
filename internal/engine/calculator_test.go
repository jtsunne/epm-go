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

// makeNodeOS returns a NodeOSStats pointer with the given CPU percent.
func makeNodeOS(cpuPercent int) *client.NodeOSStats {
	s := &client.NodeOSStats{}
	s.CPU.Percent = cpuPercent
	return s
}

// makeNodeJVM returns a NodeJVMStats pointer with the given heap values.
func makeNodeJVM(used, max int64) *client.NodeJVMStats {
	s := &client.NodeJVMStats{}
	s.Mem.HeapUsedInBytes = used
	s.Mem.HeapMaxInBytes = max
	return s
}

// makeNodeFS returns a NodeFSStats pointer with the given total/available values.
func makeNodeFS(total, available int64) *client.NodeFSStats {
	s := &client.NodeFSStats{}
	s.Total.TotalInBytes = total
	s.Total.AvailableInBytes = available
	return s
}

func TestCalcClusterResources_NilSnapshot(t *testing.T) {
	got := CalcClusterResources(nil)
	assert.Equal(t, model.ClusterResources{}, got)
}

func TestCalcClusterResources_CPUAverageSkipsZeros(t *testing.T) {
	// node1=40%, node2=0% (skipped), node3=60% → average = (40+60)/2 = 50%
	snap := &model.Snapshot{
		NodeStats: client.NodeStatsResponse{
			Nodes: map[string]client.NodePerformanceStats{
				"n1": {Name: "n1", OS: makeNodeOS(40)},
				"n2": {Name: "n2", OS: makeNodeOS(0)},
				"n3": {Name: "n3", OS: makeNodeOS(60)},
			},
		},
	}
	got := CalcClusterResources(snap)
	assert.InDelta(t, 50.0, got.AvgCPUPercent, 1e-9)
}

func TestCalcClusterResources_JVMHeapPercentage(t *testing.T) {
	// node1: 3GB used / 4GB max = 75%; node2: 1GB used / 2GB max = 50%
	// average = (75 + 50) / 2 = 62.5%
	const gb = int64(1 << 30)
	snap := &model.Snapshot{
		NodeStats: client.NodeStatsResponse{
			Nodes: map[string]client.NodePerformanceStats{
				"n1": {Name: "n1", JVM: makeNodeJVM(3*gb, 4*gb)},
				"n2": {Name: "n2", JVM: makeNodeJVM(gb, 2*gb)},
			},
		},
	}
	got := CalcClusterResources(snap)
	assert.InDelta(t, 62.5, got.AvgJVMHeapPercent, 1e-6)
}

func TestCalcClusterResources_JVMSkipsZeroHeap(t *testing.T) {
	// node1: 75%; node2: heap_max=0 → skipped
	const gb = int64(1 << 30)
	snap := &model.Snapshot{
		NodeStats: client.NodeStatsResponse{
			Nodes: map[string]client.NodePerformanceStats{
				"n1": {Name: "n1", JVM: makeNodeJVM(3*gb, 4*gb)},
				"n2": {Name: "n2", JVM: makeNodeJVM(0, 0)},
			},
		},
	}
	got := CalcClusterResources(snap)
	assert.InDelta(t, 75.0, got.AvgJVMHeapPercent, 1e-6)
}

func TestCalcClusterResources_StorageSumAndPercent(t *testing.T) {
	// node1: 100GB total, 20GB available → 80GB used
	// node2: 200GB total, 50GB available → 150GB used
	// total = 300GB, used = 230GB, percent = 230/300*100 ≈ 76.67%
	const gb = int64(1 << 30)
	snap := &model.Snapshot{
		NodeStats: client.NodeStatsResponse{
			Nodes: map[string]client.NodePerformanceStats{
				"n1": {Name: "n1", FS: makeNodeFS(100*gb, 20*gb)},
				"n2": {Name: "n2", FS: makeNodeFS(200*gb, 50*gb)},
			},
		},
	}
	got := CalcClusterResources(snap)
	assert.Equal(t, 300*gb, got.StorageTotalBytes)
	assert.Equal(t, 230*gb, got.StorageUsedBytes)
	assert.InDelta(t, 230.0/300.0*100, got.StoragePercent, 1e-9)
}

func TestCalcClusterResources_NilFields(t *testing.T) {
	// Node with all nil fields: should not panic, all metrics stay zero.
	snap := &model.Snapshot{
		NodeStats: client.NodeStatsResponse{
			Nodes: map[string]client.NodePerformanceStats{
				"n1": {Name: "n1", OS: nil, JVM: nil, FS: nil},
			},
		},
	}
	got := CalcClusterResources(snap)
	assert.Equal(t, 0.0, got.AvgCPUPercent)
	assert.Equal(t, 0.0, got.AvgJVMHeapPercent)
	assert.Equal(t, int64(0), got.StorageTotalBytes)
	assert.Equal(t, int64(0), got.StorageUsedBytes)
	assert.Equal(t, 0.0, got.StoragePercent)
}
