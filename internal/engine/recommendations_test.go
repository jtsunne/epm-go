package engine

import (
	"strings"
	"testing"

	"github.com/jtsunne/epm-go/internal/client"
	"github.com/jtsunne/epm-go/internal/model"
	"github.com/stretchr/testify/assert"
)

// makeSnap builds a minimal Snapshot for recommendation tests.
func makeSnap(status string, activeShards, unassignedShards int) *model.Snapshot {
	return &model.Snapshot{
		Health: client.ClusterHealth{
			Status:           status,
			ActiveShards:     activeShards,
			UnassignedShards: unassignedShards,
		},
	}
}

// hasRec returns true if any recommendation in recs has the given severity and
// contains titleSubstr in its Title field.
func hasRec(recs []model.Recommendation, sev model.RecommendationSeverity, titleSubstr string) bool {
	for _, r := range recs {
		if r.Severity == sev && strings.Contains(r.Title, titleSubstr) {
			return true
		}
	}
	return false
}

// hasRecCategory returns true if any recommendation in recs matches severity and category.
func hasRecCategory(recs []model.Recommendation, sev model.RecommendationSeverity, cat model.RecommendationCategory) bool {
	for _, r := range recs {
		if r.Severity == sev && r.Category == cat {
			return true
		}
	}
	return false
}

func TestCalcRecommendations_NilSnap(t *testing.T) {
	recs := CalcRecommendations(nil, model.ClusterResources{}, nil, nil)
	assert.NotNil(t, recs, "must return non-nil slice")
	assert.Empty(t, recs)
}

func TestCalcRecommendations_HealthyCluster(t *testing.T) {
	snap := makeSnap("green", 10, 0)
	resources := model.ClusterResources{
		TotalHeapMaxBytes: 8 * oneGiBInt64,
		AvgCPUPercent:     30,
		AvgJVMHeapPercent: 50,
		StoragePercent:    40,
	}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.Empty(t, recs, "healthy cluster should produce no recommendations")
}

func TestCalcRecommendations_ClusterStatusRed(t *testing.T) {
	snap := makeSnap("red", 10, 0)
	recs := CalcRecommendations(snap, model.ClusterResources{}, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityCritical, "RED"), "expect critical RED status recommendation")
}

func TestCalcRecommendations_ClusterStatusYellow(t *testing.T) {
	snap := makeSnap("yellow", 10, 0)
	recs := CalcRecommendations(snap, model.ClusterResources{}, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityWarning, "YELLOW"), "expect warning YELLOW status recommendation")
}

func TestCalcRecommendations_UnassignedShards(t *testing.T) {
	snap := makeSnap("yellow", 10, 5)
	recs := CalcRecommendations(snap, model.ClusterResources{}, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityCritical, "Unassigned shards"), "expect critical for unassigned shards")
	// Detail should mention the count.
	for _, r := range recs {
		if strings.Contains(r.Title, "Unassigned shards") {
			assert.Contains(t, r.Detail, "5 unassigned")
		}
	}
}

// Shard-to-heap: small RAM (4 GB heap, 200 shards) → critical (50/GB > 40).
func TestCalcRecommendations_ShardHeapRatio_SmallRAM_Critical(t *testing.T) {
	snap := makeSnap("green", 200, 0)
	resources := model.ClusterResources{
		TotalHeapMaxBytes: 4 * oneGiBInt64,
	}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityCritical, "shards per GB heap (critical)"),
		"4 GB heap with 200 shards (50/GB) must be critical")
}

// Shard-to-heap: small RAM (4 GB heap, 100 shards) → warning (25/GB > 20).
func TestCalcRecommendations_ShardHeapRatio_SmallRAM_Warning(t *testing.T) {
	snap := makeSnap("green", 100, 0)
	resources := model.ClusterResources{
		TotalHeapMaxBytes: 4 * oneGiBInt64,
	}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityWarning, "shards per GB heap"),
		"4 GB heap with 100 shards (25/GB) must be warning")
	assert.False(t, hasRec(recs, model.SeverityCritical, "shards per GB heap (critical)"),
		"should not be critical at 25/GB")
}

// Shard-to-heap: large RAM (64 GB heap, 200 shards) → no recommendation (3.1/GB < 20).
func TestCalcRecommendations_ShardHeapRatio_LargeRAM_Fine(t *testing.T) {
	snap := makeSnap("green", 200, 0)
	resources := model.ClusterResources{
		TotalHeapMaxBytes: 64 * oneGiBInt64,
	}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.False(t, hasRec(recs, model.SeverityCritical, "shards per GB heap"),
		"64 GB heap with 200 shards is well within limits")
	assert.False(t, hasRec(recs, model.SeverityWarning, "shards per GB heap"),
		"64 GB heap with 200 shards is well within limits")
}

func TestCalcRecommendations_CPUCritical(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	resources := model.ClusterResources{AvgCPUPercent: 95}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityCritical, "CPU pressure"))
}

func TestCalcRecommendations_CPUWarning(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	resources := model.ClusterResources{AvgCPUPercent: 85}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityWarning, "CPU usage"))
	assert.False(t, hasRec(recs, model.SeverityCritical, "CPU pressure"))
}

func TestCalcRecommendations_JVMCritical(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	resources := model.ClusterResources{AvgJVMHeapPercent: 90, TotalHeapMaxBytes: 8 * oneGiBInt64}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityCritical, "JVM heap pressure"))
	// Detail should mention total heap GB.
	for _, r := range recs {
		if strings.Contains(r.Title, "JVM heap pressure") {
			assert.Contains(t, r.Detail, "8.0 GB total heap")
		}
	}
}

func TestCalcRecommendations_JVMWarning(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	resources := model.ClusterResources{AvgJVMHeapPercent: 80, TotalHeapMaxBytes: 16 * oneGiBInt64}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityWarning, "JVM heap usage"))
}

func TestCalcRecommendations_StorageCritical(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	resources := model.ClusterResources{StoragePercent: 92}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityCritical, "storage usage"))
}

func TestCalcRecommendations_StorageWarning(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	resources := model.ClusterResources{StoragePercent: 85}
	recs := CalcRecommendations(snap, resources, nil, nil)
	assert.True(t, hasRec(recs, model.SeverityWarning, "storage usage"))
}

func TestCalcRecommendations_ZeroReplicaIndices(t *testing.T) {
	snap := makeSnap("green", 5, 0)
	indexRows := []model.IndexRow{
		{Name: "myindex", PrimaryShards: 3, TotalShards: 3, RepKnown: true},  // no replicas (rep=0)
		{Name: "other", PrimaryShards: 2, TotalShards: 4, RepKnown: true},    // has replicas
		{Name: ".system", PrimaryShards: 1, TotalShards: 1, RepKnown: true},  // system — excluded
		{Name: "closed", PrimaryShards: 1, TotalShards: 1, RepKnown: false},  // rep="-", must not count
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nil, indexRows)
	assert.True(t, hasRec(recs, model.SeverityWarning, "without replicas"))
	for _, r := range recs {
		if strings.Contains(r.Title, "without replicas") {
			assert.Contains(t, r.Detail, "1 index has") // singular: only myindex, not .system
		}
	}
}

func TestCalcRecommendations_OversizedShards(t *testing.T) {
	snap := makeSnap("green", 1, 0)
	// 1 shard, 60 GB total → avg 60 GB > 50 GB threshold.
	indexRows := []model.IndexRow{
		{Name: "bigindex", PrimaryShards: 1, TotalShards: 1, TotalSizeBytes: 60 * oneGiBInt64},
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nil, indexRows)
	assert.True(t, hasRec(recs, model.SeverityWarning, "Oversized shards"))
}

func TestCalcRecommendations_OverSharding(t *testing.T) {
	snap := makeSnap("green", 110, 0)
	// 110 shards, each 100 MB → avg < 1GB, 110 shards > 10*1 (1 data node).
	shardBytes := int64(100) * oneMiBInt64
	indexRows := make([]model.IndexRow, 110)
	for i := range indexRows {
		indexRows[i] = model.IndexRow{
			Name:          "idx",
			PrimaryShards: 1,
			TotalShards:   1,
			TotalSizeBytes: shardBytes,
		}
	}
	nodeRows := []model.NodeRow{{Name: "node1", Role: "d"}}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, indexRows)
	assert.True(t, hasRec(recs, model.SeverityWarning, "Over-sharding"))
}

// Data-to-heap ratio > 30 triggers warning.
func TestCalcRecommendations_DataToHeapRatio(t *testing.T) {
	snap := makeSnap("green", 10, 0)
	// 4 GB heap, 200 GB data → ratio = 50 > 30.
	resources := model.ClusterResources{
		TotalHeapMaxBytes: 4 * oneGiBInt64,
	}
	indexRows := []model.IndexRow{
		{Name: "idx", PrimaryShards: 1, TotalShards: 1, TotalSizeBytes: 200 * oneGiBInt64},
	}
	recs := CalcRecommendations(snap, resources, nil, indexRows)
	assert.True(t, hasRec(recs, model.SeverityWarning, "data-to-heap ratio"))
}

// Data-to-heap ratio at 25 (under threshold) produces no warning.
func TestCalcRecommendations_DataToHeapRatio_Fine(t *testing.T) {
	snap := makeSnap("green", 10, 0)
	// 4 GB heap, 100 GB data → ratio = 25 < 30.
	resources := model.ClusterResources{
		TotalHeapMaxBytes: 4 * oneGiBInt64,
	}
	indexRows := []model.IndexRow{
		{Name: "idx", PrimaryShards: 1, TotalShards: 1, TotalSizeBytes: 100 * oneGiBInt64},
	}
	recs := CalcRecommendations(snap, resources, nil, indexRows)
	assert.False(t, hasRec(recs, model.SeverityWarning, "data-to-heap ratio"))
}

func TestCalcRecommendations_SingleDataNode(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	nodeRows := []model.NodeRow{
		{Name: "node1", Role: "d"},
		{Name: "master1", Role: "m"},
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, nil)
	assert.True(t, hasRec(recs, model.SeverityWarning, "Single data node"))
}

func TestCalcRecommendations_TwoDataNodes_NoSPOF(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	nodeRows := []model.NodeRow{
		{Name: "node1", Role: "d"},
		{Name: "node2", Role: "d"},
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, nil)
	assert.False(t, hasRec(recs, model.SeverityWarning, "Single data node"))
}

// Heap hotspot: spread > 30% triggers warning.
func TestCalcRecommendations_HeapHotspot(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	nodeRows := []model.NodeRow{
		{Name: "node1", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64 * 9 / 10},  // 90%
		{Name: "node2", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64 * 5 / 10},  // 50%
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, nil)
	assert.True(t, hasRec(recs, model.SeverityWarning, "heap utilization"))
}

// Heap hotspot: spread ≤ 30% produces no hotspot warning.
func TestCalcRecommendations_HeapHotspot_Fine(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	nodeRows := []model.NodeRow{
		{Name: "node1", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64 * 7 / 10},  // 70%
		{Name: "node2", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64 * 5 / 10},  // 50%
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, nil)
	assert.False(t, hasRec(recs, model.SeverityWarning, "heap utilization"))
}

// Single node: hotspot check skipped (needs >= 2 nodes).
func TestCalcRecommendations_HeapHotspot_SingleNode(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	nodeRows := []model.NodeRow{
		{Name: "node1", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64},
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, nil)
	assert.False(t, hasRec(recs, model.SeverityWarning, "heap utilization"))
}

func TestCalcRecommendations_HotspotDetail(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	nodeRows := []model.NodeRow{
		{Name: "node1", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64 * 95 / 100}, // 95%
		{Name: "node2", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64 * 40 / 100}, // 40%
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, nil)
	for _, r := range recs {
		if strings.Contains(r.Title, "heap utilization") {
			assert.Contains(t, r.Detail, "high: 95%")
			assert.Contains(t, r.Detail, "low: 40%")
		}
	}
}

func TestCalcRecommendations_AllCategories(t *testing.T) {
	// Verify all four categories can appear.
	snap := makeSnap("yellow", 200, 3)
	resources := model.ClusterResources{
		TotalHeapMaxBytes: 4 * oneGiBInt64,
		AvgCPUPercent:     85,
	}
	indexRows := []model.IndexRow{
		{Name: "idx", PrimaryShards: 2, TotalShards: 2, RepKnown: true, TotalSizeBytes: 200 * oneMiBInt64},
	}
	nodeRows := []model.NodeRow{
		{Name: "n1", Role: "d", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64 * 9 / 10},
		{Name: "n2", Role: "d", HeapMaxBytes: oneGiBInt64, HeapUsedBytes: oneGiBInt64 * 4 / 10},
	}
	recs := CalcRecommendations(snap, resources, nodeRows, indexRows)

	assert.True(t, hasRecCategory(recs, model.SeverityWarning, model.CategoryResourcePressure), "ResourcePressure")
	assert.True(t, hasRecCategory(recs, model.SeverityCritical, model.CategoryShardHealth), "ShardHealth critical (unassigned)")
	assert.True(t, hasRecCategory(recs, model.SeverityWarning, model.CategoryIndexConfig), "IndexConfig (zero-replica)")
	assert.True(t, hasRecCategory(recs, model.SeverityWarning, model.CategoryHotspot), "Hotspot")
}

// TestCalcRecommendations_TieredDataNodes verifies that ES 8.x+ tiered data role
// abbreviations ('h'=data_hot, 'w'=data_warm, 'c'=data_cold, 'f'=data_frozen,
// 's'=data_content) are counted as data nodes, preventing false SPOF warnings.
func TestCalcRecommendations_TieredDataNodes_NoSPOF(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	// Two hot-tier nodes — no generic 'd' role, but should not trigger SPOF.
	nodeRows := []model.NodeRow{
		{Name: "hot1", Role: "h"},
		{Name: "hot2", Role: "h"},
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, nil)
	assert.False(t, hasRec(recs, model.SeverityWarning, "Single data node"), "two hot-tier nodes must not trigger SPOF")
}

func TestCalcRecommendations_TieredDataNodes_SingleNode_SPOF(t *testing.T) {
	snap := makeSnap("green", 0, 0)
	// Only one warm-tier node — should trigger SPOF.
	nodeRows := []model.NodeRow{
		{Name: "warm1", Role: "w"},
		{Name: "master1", Role: "m"},
	}
	recs := CalcRecommendations(snap, model.ClusterResources{}, nodeRows, nil)
	assert.True(t, hasRec(recs, model.SeverityWarning, "Single data node"), "single warm-tier node must trigger SPOF")
}
