package tui

import (
	"testing"

	"github.com/jtsunne/epm-go/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indexRowFixtures returns a reproducible set of IndexRow test data.
func indexRowFixtures() []model.IndexRow {
	return []model.IndexRow{
		{Name: "logs-2024", IndexingRate: 100, SearchRate: 50, IndexLatency: 5, SearchLatency: 10, TotalSizeBytes: 1000, AvgShardSize: 200, DocCount: 500, PrimaryShards: 1, TotalShards: 2},
		{Name: "app-events", IndexingRate: 300, SearchRate: 200, IndexLatency: 2, SearchLatency: 4, TotalSizeBytes: 3000, AvgShardSize: 600, DocCount: 1500, PrimaryShards: 2, TotalShards: 4},
		{Name: "metrics", IndexingRate: 50, SearchRate: 10, IndexLatency: 8, SearchLatency: 20, TotalSizeBytes: 500, AvgShardSize: 100, DocCount: 250, PrimaryShards: 1, TotalShards: 1},
		{Name: "Audit-Logs", IndexingRate: 150, SearchRate: 75, IndexLatency: 3, SearchLatency: 6, TotalSizeBytes: 1500, AvgShardSize: 300, DocCount: 750, PrimaryShards: 3, TotalShards: 6},
	}
}

// nodeRowFixtures returns a reproducible set of NodeRow test data.
func nodeRowFixtures() []model.NodeRow {
	return []model.NodeRow{
		{Name: "node-1", Role: "master", IP: "10.0.0.1", IndexingRate: 200, SearchRate: 100, IndexLatency: 3, SearchLatency: 6},
		{Name: "node-2", Role: "data", IP: "10.0.0.2", IndexingRate: 500, SearchRate: 250, IndexLatency: 1, SearchLatency: 2},
		{Name: "node-3", Role: "data", IP: "192.168.1.10", IndexingRate: 100, SearchRate: 50, IndexLatency: 5, SearchLatency: 10},
	}
}

// ---------- sortIndexRows ----------

func TestSortIndexRows_ByRate(t *testing.T) {
	rows := indexRowFixtures()
	sorted := sortIndexRows(rows, 5, true) // col 5 = IndexingRate, descending
	require.Len(t, sorted, 4)
	assert.Equal(t, "app-events", sorted[0].Name) // 300
	assert.Equal(t, "Audit-Logs", sorted[1].Name) // 150
	assert.Equal(t, "logs-2024", sorted[2].Name)  // 100
	assert.Equal(t, "metrics", sorted[3].Name)    // 50
}

func TestSortIndexRows_ByRate_Ascending(t *testing.T) {
	rows := indexRowFixtures()
	sorted := sortIndexRows(rows, 5, false) // ascending
	require.Len(t, sorted, 4)
	assert.Equal(t, "metrics", sorted[0].Name)    // 50
	assert.Equal(t, "logs-2024", sorted[1].Name)  // 100
	assert.Equal(t, "Audit-Logs", sorted[2].Name) // 150
	assert.Equal(t, "app-events", sorted[3].Name) // 300
}

func TestSortIndexRows_ByName(t *testing.T) {
	rows := indexRowFixtures()
	sorted := sortIndexRows(rows, 0, false) // col 0 = Name, ascending (case-insensitive)
	require.Len(t, sorted, 4)
	assert.Equal(t, "app-events", sorted[0].Name)
	assert.Equal(t, "Audit-Logs", sorted[1].Name)
	assert.Equal(t, "logs-2024", sorted[2].Name)
	assert.Equal(t, "metrics", sorted[3].Name)
}

func TestSortIndexRows_ByName_Descending(t *testing.T) {
	rows := indexRowFixtures()
	sorted := sortIndexRows(rows, 0, true)
	require.Len(t, sorted, 4)
	assert.Equal(t, "metrics", sorted[0].Name)
	assert.Equal(t, "logs-2024", sorted[1].Name)
	assert.Equal(t, "Audit-Logs", sorted[2].Name)
	assert.Equal(t, "app-events", sorted[3].Name)
}

func TestSortIndexRows_ToggleDirection(t *testing.T) {
	rows := indexRowFixtures()
	asc := sortIndexRows(rows, 5, false)
	desc := sortIndexRows(rows, 5, true)
	require.Len(t, asc, 4)
	require.Len(t, desc, 4)
	assert.Equal(t, asc[0].Name, desc[len(desc)-1].Name)
	assert.Equal(t, asc[len(asc)-1].Name, desc[0].Name)
}

func TestSortIndexRows_NoSort(t *testing.T) {
	rows := indexRowFixtures()
	result := sortIndexRows(rows, -1, true)
	require.Len(t, result, 4)
	// Order preserved
	assert.Equal(t, rows[0].Name, result[0].Name)
	assert.Equal(t, rows[1].Name, result[1].Name)
}

func TestSortIndexRows_DoesNotMutateInput(t *testing.T) {
	rows := indexRowFixtures()
	original := make([]model.IndexRow, len(rows))
	copy(original, rows)
	sortIndexRows(rows, 5, true)
	assert.Equal(t, original, rows)
}

func TestSortIndexRows_BySearchRate(t *testing.T) {
	rows := indexRowFixtures()
	sorted := sortIndexRows(rows, 6, true) // col 6 = SearchRate
	assert.Equal(t, "app-events", sorted[0].Name) // 200
}

func TestSortIndexRows_ByTotalSize(t *testing.T) {
	rows := indexRowFixtures()
	sorted := sortIndexRows(rows, 2, true) // col 2 = TotalSizeBytes
	assert.Equal(t, "app-events", sorted[0].Name) // 3000
}

func TestSortIndexRows_ByDocCount(t *testing.T) {
	rows := indexRowFixtures()
	sorted := sortIndexRows(rows, 4, false) // col 4 = DocCount ascending
	assert.Equal(t, "metrics", sorted[0].Name) // 250
}

func TestSortIndexRows_SentinelLastAscending(t *testing.T) {
	// MetricNotAvailable (-1.0) sentinel rows must sort last even in ascending order.
	rows := []model.IndexRow{
		{Name: "b", IndexingRate: model.MetricNotAvailable},
		{Name: "a", IndexingRate: 100},
		{Name: "c", IndexingRate: 50},
	}
	sorted := sortIndexRows(rows, 5, false) // ascending
	require.Len(t, sorted, 3)
	assert.Equal(t, "c", sorted[0].Name)   // 50 first
	assert.Equal(t, "a", sorted[1].Name)   // 100 second
	assert.Equal(t, "b", sorted[2].Name)   // sentinel last
}

func TestSortIndexRows_SentinelLastDescending(t *testing.T) {
	rows := []model.IndexRow{
		{Name: "b", IndexingRate: model.MetricNotAvailable},
		{Name: "a", IndexingRate: 100},
		{Name: "c", IndexingRate: 50},
	}
	sorted := sortIndexRows(rows, 5, true) // descending
	require.Len(t, sorted, 3)
	assert.Equal(t, "a", sorted[0].Name)   // 100 first
	assert.Equal(t, "c", sorted[1].Name)   // 50 second
	assert.Equal(t, "b", sorted[2].Name)   // sentinel last
}

// ---------- filterIndexRows ----------

func TestFilterIndexRows_CaseInsensitive(t *testing.T) {
	rows := indexRowFixtures()

	result := filterIndexRows(rows, "LOGS")
	require.Len(t, result, 2)
	names := []string{result[0].Name, result[1].Name}
	assert.Contains(t, names, "logs-2024")
	assert.Contains(t, names, "Audit-Logs")
}

func TestFilterIndexRows_ExactPrefix(t *testing.T) {
	rows := indexRowFixtures()
	result := filterIndexRows(rows, "app")
	require.Len(t, result, 1)
	assert.Equal(t, "app-events", result[0].Name)
}

func TestFilterIndexRows_EmptySearch(t *testing.T) {
	rows := indexRowFixtures()
	result := filterIndexRows(rows, "")
	assert.Len(t, result, len(rows))
}

func TestFilterIndexRows_NoMatch(t *testing.T) {
	rows := indexRowFixtures()
	result := filterIndexRows(rows, "xyzzy")
	assert.Len(t, result, 0)
}

func TestFilterIndexRows_SpecialChars(t *testing.T) {
	rows := indexRowFixtures()
	// logs-2024, app-events, Audit-Logs all contain "-"
	result := filterIndexRows(rows, "-")
	require.Len(t, result, 3)
}

// ---------- sortNodeRows ----------

func TestSortNodeRows_BySearchRate(t *testing.T) {
	rows := nodeRowFixtures()
	sorted := sortNodeRows(rows, 4, true) // col 4 = SearchRate, descending
	require.Len(t, sorted, 3)
	assert.Equal(t, "node-2", sorted[0].Name) // 250
	assert.Equal(t, "node-1", sorted[1].Name) // 100
	assert.Equal(t, "node-3", sorted[2].Name) // 50
}

func TestSortNodeRows_ByIndexingRate(t *testing.T) {
	rows := nodeRowFixtures()
	sorted := sortNodeRows(rows, 3, true) // col 3 = IndexingRate desc
	assert.Equal(t, "node-2", sorted[0].Name) // 500
}

func TestSortNodeRows_ByName(t *testing.T) {
	rows := nodeRowFixtures()
	sorted := sortNodeRows(rows, 0, false)
	assert.Equal(t, "node-1", sorted[0].Name)
	assert.Equal(t, "node-2", sorted[1].Name)
	assert.Equal(t, "node-3", sorted[2].Name)
}

func TestSortNodeRows_DoesNotMutateInput(t *testing.T) {
	rows := nodeRowFixtures()
	original := make([]model.NodeRow, len(rows))
	copy(original, rows)
	sortNodeRows(rows, 3, true)
	assert.Equal(t, original, rows)
}

func TestSortNodeRows_SentinelLastAscending(t *testing.T) {
	rows := []model.NodeRow{
		{Name: "node-b", IndexingRate: model.MetricNotAvailable},
		{Name: "node-a", IndexingRate: 200},
		{Name: "node-c", IndexingRate: 50},
	}
	sorted := sortNodeRows(rows, 3, false) // col 3 = IndexingRate ascending
	require.Len(t, sorted, 3)
	assert.Equal(t, "node-c", sorted[0].Name)  // 50 first
	assert.Equal(t, "node-a", sorted[1].Name)  // 200 second
	assert.Equal(t, "node-b", sorted[2].Name)  // sentinel last
}

func TestSortNodeRows_ByShards(t *testing.T) {
	rows := []model.NodeRow{
		{Name: "node-a", Shards: 10},
		{Name: "node-b", Shards: 5},
		{Name: "node-c", Shards: 20},
	}
	sorted := sortNodeRows(rows, 7, true) // descending
	require.Len(t, sorted, 3)
	assert.Equal(t, "node-c", sorted[0].Name) // 20
	assert.Equal(t, "node-a", sorted[1].Name) // 10
	assert.Equal(t, "node-b", sorted[2].Name) // 5
}

func TestSortNodeRows_ByDiskPercent(t *testing.T) {
	rows := []model.NodeRow{
		{Name: "node-a", DiskPercent: 30.0},
		{Name: "node-b", DiskPercent: 80.0},
		{Name: "node-c", DiskPercent: 55.0},
	}
	sorted := sortNodeRows(rows, 8, false) // ascending
	require.Len(t, sorted, 3)
	assert.Equal(t, "node-a", sorted[0].Name) // 30.0
	assert.Equal(t, "node-c", sorted[1].Name) // 55.0
	assert.Equal(t, "node-b", sorted[2].Name) // 80.0
}

func TestSortNodeRows_ShardsSentinelLast(t *testing.T) {
	rows := []model.NodeRow{
		{Name: "node-b", Shards: -1}, // sentinel
		{Name: "node-a", Shards: 10},
		{Name: "node-c", Shards: 5},
	}
	sorted := sortNodeRows(rows, 7, false) // ascending
	require.Len(t, sorted, 3)
	assert.Equal(t, "node-c", sorted[0].Name)  // 5 first
	assert.Equal(t, "node-a", sorted[1].Name)  // 10 second
	assert.Equal(t, "node-b", sorted[2].Name)  // sentinel last
}

func TestSortNodeRows_DiskPercentSentinelLast(t *testing.T) {
	rows := []model.NodeRow{
		{Name: "node-b", DiskPercent: -1.0}, // sentinel
		{Name: "node-a", DiskPercent: 60.0},
		{Name: "node-c", DiskPercent: 30.0},
	}
	sorted := sortNodeRows(rows, 8, false) // ascending
	require.Len(t, sorted, 3)
	assert.Equal(t, "node-c", sorted[0].Name)  // 30.0 first
	assert.Equal(t, "node-a", sorted[1].Name)  // 60.0 second
	assert.Equal(t, "node-b", sorted[2].Name)  // sentinel last
}

// ---------- filterNodeRows ----------

func TestFilterNodeRows_ByIP(t *testing.T) {
	rows := nodeRowFixtures()
	result := filterNodeRows(rows, "192.168")
	require.Len(t, result, 1)
	assert.Equal(t, "node-3", result[0].Name)
}

func TestFilterNodeRows_ByName(t *testing.T) {
	rows := nodeRowFixtures()
	result := filterNodeRows(rows, "node-1")
	require.Len(t, result, 1)
	assert.Equal(t, "node-1", result[0].Name)
}

func TestFilterNodeRows_ByNameCaseInsensitive(t *testing.T) {
	rows := nodeRowFixtures()
	result := filterNodeRows(rows, "NODE")
	assert.Len(t, result, 3)
}

func TestFilterNodeRows_EmptySearch(t *testing.T) {
	rows := nodeRowFixtures()
	result := filterNodeRows(rows, "")
	assert.Len(t, result, len(rows))
}

func TestFilterNodeRows_NoMatch(t *testing.T) {
	rows := nodeRowFixtures()
	result := filterNodeRows(rows, "xyzzy")
	assert.Len(t, result, 0)
}

func TestFilterNodeRows_IPSubnet(t *testing.T) {
	rows := nodeRowFixtures()
	result := filterNodeRows(rows, "10.0.0")
	require.Len(t, result, 2)
}
