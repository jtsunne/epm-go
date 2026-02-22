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

// makeClusterIndexStats builds a single-index IndexStatsResponse for testing
// CalcClusterMetrics. Indexing ops are placed in Primaries; search ops in Total.
func makeClusterIndexStats(idxOps, idxTimeMs, srchOps, srchTimeMs int64) client.IndexStatsResponse {
	return client.IndexStatsResponse{
		Indices: map[string]client.IndexStatEntry{
			"idx": makeIndexStats(idxOps, idxTimeMs, -1, -1, -1, -1, srchOps, srchTimeMs, -1, -1),
		},
	}
}

func TestCalcClusterMetrics_FirstSnapshot(t *testing.T) {
	curr := &model.Snapshot{
		IndexStats: makeClusterIndexStats(1000, 500, 2000, 800),
	}
	got := CalcClusterMetrics(nil, curr, 10*time.Second)
	want := model.PerformanceMetrics{
		IndexingRate:  model.MetricNotAvailable,
		SearchRate:    model.MetricNotAvailable,
		IndexLatency:  model.MetricNotAvailable,
		SearchLatency: model.MetricNotAvailable,
	}
	assert.Equal(t, want, got)
}

func TestCalcClusterMetrics_NilCurr(t *testing.T) {
	prev := &model.Snapshot{
		IndexStats: makeClusterIndexStats(1000, 500, 2000, 800),
	}
	got := CalcClusterMetrics(prev, nil, 10*time.Second)
	want := model.PerformanceMetrics{
		IndexingRate:  model.MetricNotAvailable,
		SearchRate:    model.MetricNotAvailable,
		IndexLatency:  model.MetricNotAvailable,
		SearchLatency: model.MetricNotAvailable,
	}
	assert.Equal(t, want, got)
}

func TestCalcClusterMetrics_BasicRates(t *testing.T) {
	// prev: 1000 index ops (primaries), 500ms; 2000 search ops (total), 800ms
	// curr: 2000 index ops (primaries), 700ms; 3500 search ops (total), 1300ms
	// elapsed: 10s
	// indexingRate = (2000-1000)/10 = 100 /s
	// searchRate   = (3500-2000)/10 = 150 /s
	// indexLatency = (700-500)/(2000-1000) = 200/1000 = 0.2 ms
	// searchLatency = (1300-800)/(3500-2000) = 500/1500 ≈ 0.333 ms
	prev := &model.Snapshot{
		IndexStats: makeClusterIndexStats(1000, 500, 2000, 800),
	}
	curr := &model.Snapshot{
		IndexStats: makeClusterIndexStats(2000, 700, 3500, 1300),
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
		IndexStats: makeClusterIndexStats(5000, 2000, 8000, 3000),
	}
	curr := &model.Snapshot{
		IndexStats: makeClusterIndexStats(100, 50, 200, 80),
	}
	got := CalcClusterMetrics(prev, curr, 10*time.Second)
	assert.Equal(t, 0.0, got.IndexingRate)
	assert.Equal(t, 0.0, got.SearchRate)
}

func TestCalcClusterMetrics_TooShortInterval(t *testing.T) {
	prev := &model.Snapshot{
		IndexStats: makeClusterIndexStats(1000, 500, 2000, 800),
	}
	curr := &model.Snapshot{
		IndexStats: makeClusterIndexStats(2000, 700, 3500, 1300),
	}
	// 500ms < 1s → return sentinels
	got := CalcClusterMetrics(prev, curr, 500*time.Millisecond)
	want := model.PerformanceMetrics{
		IndexingRate:  model.MetricNotAvailable,
		SearchRate:    model.MetricNotAvailable,
		IndexLatency:  model.MetricNotAvailable,
		SearchLatency: model.MetricNotAvailable,
	}
	assert.Equal(t, want, got)
}

func TestCalcClusterMetrics_RateSanityCap(t *testing.T) {
	// delta = maxRatePerSec*10+1 ops in 1 second → rate exceeds cap → clamped to 0
	bigDelta := int64(maxRatePerSec*10 + 1)
	prev := &model.Snapshot{
		IndexStats: makeClusterIndexStats(0, 0, 0, 0),
	}
	curr := &model.Snapshot{
		IndexStats: makeClusterIndexStats(bigDelta, 0, bigDelta, 0),
	}
	got := CalcClusterMetrics(prev, curr, 1*time.Second)
	assert.Equal(t, 0.0, got.IndexingRate)
	assert.Equal(t, 0.0, got.SearchRate)
}

func TestCalcClusterMetrics_IndexDisappears(t *testing.T) {
	// prev has two indices: "big" with 100,000 ops and "small" with 200 ops.
	// curr has only "small" with 250 ops ("big" was deleted/rolled over).
	// Without the fix the aggregate prev total (100,200) > curr total (250), so the
	// delta goes negative and gets clamped to 0 — masking real activity on "small".
	// With the fix only matching indices are compared: delta = 250-200 = 50 ops.
	prev := &model.Snapshot{
		IndexStats: client.IndexStatsResponse{
			Indices: map[string]client.IndexStatEntry{
				"big":   makeIndexStats(100_000, 50_000, -1, -1, -1, -1, 200_000, 100_000, -1, -1),
				"small": makeIndexStats(200, 100, -1, -1, -1, -1, 400, 200, -1, -1),
			},
		},
	}
	curr := &model.Snapshot{
		IndexStats: client.IndexStatsResponse{
			Indices: map[string]client.IndexStatEntry{
				"small": makeIndexStats(250, 130, -1, -1, -1, -1, 500, 250, -1, -1),
			},
		},
	}
	got := CalcClusterMetrics(prev, curr, 10*time.Second)
	// indexingRate = (250-200)/10 = 5/s
	assert.InDelta(t, 5.0, got.IndexingRate, 1e-9)
	// searchRate = (500-400)/10 = 10/s
	assert.InDelta(t, 10.0, got.SearchRate, 1e-9)
}

func TestCalcClusterMetrics_LatencySanityCap(t *testing.T) {
	// 1 op with enormous time → raw latency >> maxLatencyMs → capped
	hugeTime := int64(maxLatencyMs*2 + 1)
	prev := &model.Snapshot{
		IndexStats: makeClusterIndexStats(0, 0, 0, 0),
	}
	curr := &model.Snapshot{
		IndexStats: makeClusterIndexStats(1, hugeTime, 1, hugeTime),
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

func TestCalcClusterResources_JVMIncludesZeroHeapUsed(t *testing.T) {
	// node1: 0B used / 4GB max = 0%; node2: 2GB used / 4GB max = 50%
	// Both nodes have valid heap (max > 0) so both are included → average = 25%
	const gb = int64(1 << 30)
	snap := &model.Snapshot{
		NodeStats: client.NodeStatsResponse{
			Nodes: map[string]client.NodePerformanceStats{
				"n1": {Name: "n1", JVM: makeNodeJVM(0, 4*gb)},
				"n2": {Name: "n2", JVM: makeNodeJVM(2*gb, 4*gb)},
			},
		},
	}
	got := CalcClusterResources(snap)
	assert.InDelta(t, 25.0, got.AvgJVMHeapPercent, 1e-6)
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

// makeIndexStats builds an IndexStatEntry with given primaries and total values.
// Pass -1 to omit a section (leave it nil).
func makeIndexStats(priIdxOps, priIdxTime, priSrchOps, priSrchTime,
	totIdxOps, totIdxTime, totSrchOps, totSrchTime, priStoreBytes, totStoreBytes int64) client.IndexStatEntry {
	entry := client.IndexStatEntry{}

	if priIdxOps >= 0 || priSrchOps >= 0 || priStoreBytes >= 0 {
		entry.Primaries = &client.IndexStatShard{}
		if priIdxOps >= 0 {
			entry.Primaries.Indexing = &client.IndexingStats{
				IndexTotal:        priIdxOps,
				IndexTimeInMillis: priIdxTime,
			}
		}
		if priSrchOps >= 0 {
			entry.Primaries.Search = &client.SearchStats{
				QueryTotal:        priSrchOps,
				QueryTimeInMillis: priSrchTime,
			}
		}
		if priStoreBytes >= 0 {
			entry.Primaries.Store = &client.StoreStats{SizeInBytes: priStoreBytes}
		}
	}

	if totIdxOps >= 0 || totSrchOps >= 0 || totStoreBytes >= 0 {
		entry.Total = &client.IndexStatShard{}
		if totIdxOps >= 0 {
			entry.Total.Indexing = &client.IndexingStats{
				IndexTotal:        totIdxOps,
				IndexTimeInMillis: totIdxTime,
			}
		}
		if totSrchOps >= 0 {
			entry.Total.Search = &client.SearchStats{
				QueryTotal:        totSrchOps,
				QueryTimeInMillis: totSrchTime,
			}
		}
		if totStoreBytes >= 0 {
			entry.Total.Store = &client.StoreStats{SizeInBytes: totStoreBytes}
		}
	}

	return entry
}

func TestCalcIndexRows_NilPrev(t *testing.T) {
	// No previous snapshot → rate/latency fields set to MetricNotAvailable, rows are returned.
	curr := &model.Snapshot{
		Indices: []client.IndexInfo{
			{Index: "logs", Pri: "1", Rep: "0", DocsCount: "1000"},
		},
		IndexStats: client.IndexStatsResponse{
			Indices: map[string]client.IndexStatEntry{
				"logs": makeIndexStats(500, 100, 300, 60, 600, 120, 450, 90, 1024, 2048),
			},
		},
	}
	rows := CalcIndexRows(nil, curr, 10*time.Second)
	assert.Len(t, rows, 1)
	assert.Equal(t, "logs", rows[0].Name)
	assert.Equal(t, model.MetricNotAvailable, rows[0].IndexingRate)
	assert.Equal(t, model.MetricNotAvailable, rows[0].SearchRate)
	assert.Equal(t, model.MetricNotAvailable, rows[0].IndexLatency)
	assert.Equal(t, model.MetricNotAvailable, rows[0].SearchLatency)
}

func TestCalcIndexRows_ShardCountParsing(t *testing.T) {
	// pri="5", rep="1" → TotalShards = 5*(1+1) = 10
	curr := &model.Snapshot{
		Indices: []client.IndexInfo{
			{Index: "myidx", Pri: "5", Rep: "1", DocsCount: "42000"},
		},
		IndexStats: client.IndexStatsResponse{
			Indices: map[string]client.IndexStatEntry{
				"myidx": makeIndexStats(0, 0, 0, 0, 0, 0, 0, 0, 500*1024*1024, 1000*1024*1024),
			},
		},
	}
	rows := CalcIndexRows(nil, curr, 10*time.Second)
	assert.Len(t, rows, 1)
	assert.Equal(t, 5, rows[0].PrimaryShards)
	assert.Equal(t, 10, rows[0].TotalShards)
	assert.Equal(t, int64(42000), rows[0].DocCount)
	// AvgShardSize = primarySizeBytes / pri = 500MB / 5 = 100MB
	assert.Equal(t, int64(100*1024*1024), rows[0].AvgShardSize)
	assert.Equal(t, int64(1000*1024*1024), rows[0].TotalSizeBytes)
}

func TestCalcIndexRows_PrimariesForIndexing(t *testing.T) {
	// Indexing should use primaries counters, not total.
	// primaries: prev=100 ops, curr=200 ops → delta=100 → rate=10/s over 10s
	// total indexing: prev=500, curr=1000 → delta=500 → rate would be 50/s (wrong)
	prev := &model.Snapshot{
		Indices: []client.IndexInfo{
			{Index: "test", Pri: "1", Rep: "0", DocsCount: "0"},
		},
		IndexStats: client.IndexStatsResponse{
			Indices: map[string]client.IndexStatEntry{
				// priIdxOps=100, totIdxOps=500
				"test": makeIndexStats(100, 50, -1, -1, 500, 250, 200, 40, -1, -1),
			},
		},
	}
	curr := &model.Snapshot{
		Indices: []client.IndexInfo{
			{Index: "test", Pri: "1", Rep: "0", DocsCount: "0"},
		},
		IndexStats: client.IndexStatsResponse{
			Indices: map[string]client.IndexStatEntry{
				// priIdxOps=200, totIdxOps=1000
				"test": makeIndexStats(200, 100, -1, -1, 1000, 500, 400, 80, -1, -1),
			},
		},
	}
	rows := CalcIndexRows(prev, curr, 10*time.Second)
	assert.Len(t, rows, 1)
	// 100 primaries ops / 10s = 10 /s
	assert.InDelta(t, 10.0, rows[0].IndexingRate, 1e-9)
	// latency: deltaTime(50ms) / deltaOps(100) = 0.5 ms
	assert.InDelta(t, 0.5, rows[0].IndexLatency, 1e-9)
}

func TestCalcIndexRows_TotalForSearch(t *testing.T) {
	// Search should use total counters, not primaries.
	// total: prev=200 ops, curr=400 ops → delta=200 → rate=20/s over 10s
	// primaries search: prev=80, curr=160 → delta=80 → rate would be 8/s (wrong)
	prev := &model.Snapshot{
		Indices: []client.IndexInfo{
			{Index: "test", Pri: "1", Rep: "0", DocsCount: "0"},
		},
		IndexStats: client.IndexStatsResponse{
			Indices: map[string]client.IndexStatEntry{
				// priSrchOps=80, totSrchOps=200
				"test": makeIndexStats(-1, -1, 80, 40, -1, -1, 200, 100, -1, -1),
			},
		},
	}
	curr := &model.Snapshot{
		Indices: []client.IndexInfo{
			{Index: "test", Pri: "1", Rep: "0", DocsCount: "0"},
		},
		IndexStats: client.IndexStatsResponse{
			Indices: map[string]client.IndexStatEntry{
				// priSrchOps=160, totSrchOps=400
				"test": makeIndexStats(-1, -1, 160, 80, -1, -1, 400, 200, -1, -1),
			},
		},
	}
	rows := CalcIndexRows(prev, curr, 10*time.Second)
	assert.Len(t, rows, 1)
	// 200 total search ops / 10s = 20 /s
	assert.InDelta(t, 20.0, rows[0].SearchRate, 1e-9)
	// latency: deltaTime(100ms) / deltaOps(200) = 0.5 ms
	assert.InDelta(t, 0.5, rows[0].SearchLatency, 1e-9)
}

// makeNodeStatsWithID builds a NodeStatsResponse with a named node keyed by nodeID.
func makeNodeStatsWithID(nodeID, nodeName string, indexOps, indexTimeMs, searchOps, searchTimeMs int64) client.NodeStatsResponse {
	return client.NodeStatsResponse{
		Nodes: map[string]client.NodePerformanceStats{
			nodeID: {
				Name: nodeName,
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

func TestCalcNodeRows_NilCurr(t *testing.T) {
	rows := CalcNodeRows(nil, nil, 10*time.Second)
	assert.Nil(t, rows)
}

func TestCalcNodeRows_NilPrev(t *testing.T) {
	// No previous snapshot → rate/latency fields set to MetricNotAvailable, rows are returned.
	curr := &model.Snapshot{
		Nodes: []client.NodeInfo{
			{Name: "node-a", NodeRole: "d", IP: "10.0.0.1"},
		},
		NodeStats: makeNodeStatsWithID("id1", "node-a", 1000, 500, 2000, 800),
	}
	rows := CalcNodeRows(nil, curr, 10*time.Second)
	assert.Len(t, rows, 1)
	assert.Equal(t, "id1", rows[0].ID)
	assert.Equal(t, "node-a", rows[0].Name)
	assert.Equal(t, model.MetricNotAvailable, rows[0].IndexingRate)
	assert.Equal(t, model.MetricNotAvailable, rows[0].SearchRate)
	assert.Equal(t, model.MetricNotAvailable, rows[0].IndexLatency)
	assert.Equal(t, model.MetricNotAvailable, rows[0].SearchLatency)
}

func TestCalcNodeRows_BasicRates(t *testing.T) {
	// prev: 1000 index ops, 500ms; 2000 search ops, 800ms
	// curr: 2000 index ops, 700ms; 3500 search ops, 1300ms
	// elapsed: 10s
	// indexingRate = (2000-1000)/10 = 100 /s
	// searchRate   = (3500-2000)/10 = 150 /s
	// indexLatency = (700-500)/(2000-1000) = 0.2 ms
	// searchLatency = (1300-800)/(3500-2000) = 500/1500 ≈ 0.333 ms
	prev := &model.Snapshot{
		NodeStats: makeNodeStatsWithID("id1", "node-a", 1000, 500, 2000, 800),
	}
	curr := &model.Snapshot{
		Nodes: []client.NodeInfo{
			{Name: "node-a", NodeRole: "d", IP: "10.0.0.1"},
		},
		NodeStats: makeNodeStatsWithID("id1", "node-a", 2000, 700, 3500, 1300),
	}
	rows := CalcNodeRows(prev, curr, 10*time.Second)
	assert.Len(t, rows, 1)
	assert.InDelta(t, 100.0, rows[0].IndexingRate, 1e-9)
	assert.InDelta(t, 150.0, rows[0].SearchRate, 1e-9)
	assert.InDelta(t, 0.2, rows[0].IndexLatency, 1e-9)
	assert.InDelta(t, 500.0/1500.0, rows[0].SearchLatency, 1e-9)
}

func TestCalcNodeRows_NewNode(t *testing.T) {
	// Node in curr that didn't exist in prev → zero rates (enoughTime is true,
	// but no prev entry for this node), no crash.
	prev := &model.Snapshot{
		NodeStats: client.NodeStatsResponse{
			Nodes: map[string]client.NodePerformanceStats{},
		},
	}
	curr := &model.Snapshot{
		Nodes: []client.NodeInfo{
			{Name: "node-new", NodeRole: "m", IP: "10.0.0.5"},
		},
		NodeStats: makeNodeStatsWithID("id-new", "node-new", 500, 200, 300, 100),
	}
	rows := CalcNodeRows(prev, curr, 10*time.Second)
	assert.Len(t, rows, 1)
	assert.Equal(t, "id-new", rows[0].ID)
	// enoughTime is true (prev != nil, elapsed >= 1s) but node not in prev → rates are 0
	assert.Equal(t, 0.0, rows[0].IndexingRate)
	assert.Equal(t, 0.0, rows[0].SearchRate)
}

func TestCalcNodeRows_RoleAndIPFromNodes(t *testing.T) {
	// Role and IP must come from curr.Nodes (the _cat/nodes list), matched by name.
	curr := &model.Snapshot{
		Nodes: []client.NodeInfo{
			{Name: "data-node", NodeRole: "d", IP: "192.168.1.10"},
		},
		NodeStats: makeNodeStatsWithID("abc123", "data-node", 0, 0, 0, 0),
	}
	rows := CalcNodeRows(nil, curr, 10*time.Second)
	assert.Len(t, rows, 1)
	assert.Equal(t, "d", rows[0].Role)
	assert.Equal(t, "192.168.1.10", rows[0].IP)
}

func TestCalcNodeRows_TooShortInterval(t *testing.T) {
	// elapsed < 1s → sentinel rates (not enough time for valid delta).
	prev := &model.Snapshot{
		NodeStats: makeNodeStatsWithID("id1", "node-a", 1000, 500, 2000, 800),
	}
	curr := &model.Snapshot{
		NodeStats: makeNodeStatsWithID("id1", "node-a", 2000, 700, 3500, 1300),
	}
	rows := CalcNodeRows(prev, curr, 500*time.Millisecond)
	assert.Len(t, rows, 1)
	assert.Equal(t, model.MetricNotAvailable, rows[0].IndexingRate)
	assert.Equal(t, model.MetricNotAvailable, rows[0].SearchRate)
	assert.Equal(t, model.MetricNotAvailable, rows[0].IndexLatency)
	assert.Equal(t, model.MetricNotAvailable, rows[0].SearchLatency)
}
