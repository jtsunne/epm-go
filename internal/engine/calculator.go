package engine

import (
	"sort"
	"strconv"
	"time"

	"github.com/dm/epm-go/internal/client"
	"github.com/dm/epm-go/internal/model"
)

// CalcNodeRows computes per-node throughput and latency metrics from two
// consecutive snapshots. Ported from NodeTable.tsx lines 76-113.
//
// Role and IP are looked up from curr.Nodes by matching on node name.
// Nodes present in curr but not in prev get zero rates.
func CalcNodeRows(prev, curr *model.Snapshot, elapsed time.Duration) []model.NodeRow {
	if curr == nil {
		return nil
	}

	// Build name → NodeInfo lookup from _cat/nodes endpoint.
	nameToNode := make(map[string]client.NodeInfo, len(curr.Nodes))
	for _, n := range curr.Nodes {
		nameToNode[n.Name] = n
	}

	elapsedSec := elapsed.Seconds()
	enoughTime := prev != nil && elapsedSec >= minTimeDiffSeconds

	rows := make([]model.NodeRow, 0, len(curr.NodeStats.Nodes))
	for nodeID, node := range curr.NodeStats.Nodes {
		row := model.NodeRow{
			ID:   nodeID,
			Name: node.Name,
		}

		// Populate role and IP from _cat/nodes data.
		if info, ok := nameToNode[node.Name]; ok {
			row.Role = info.NodeRole
			row.IP = info.IP
		}

		if enoughTime {
			prevNode, hasPrev := prev.NodeStats.Nodes[nodeID]
			if hasPrev && node.Indices != nil && prevNode.Indices != nil {
				idxOpsDelta := maxFloat64(0, float64(node.Indices.Indexing.IndexTotal-prevNode.Indices.Indexing.IndexTotal))
				idxTimeDelta := maxFloat64(0, float64(node.Indices.Indexing.IndexTimeInMillis-prevNode.Indices.Indexing.IndexTimeInMillis))
				srchOpsDelta := maxFloat64(0, float64(node.Indices.Search.QueryTotal-prevNode.Indices.Search.QueryTotal))
				srchTimeDelta := maxFloat64(0, float64(node.Indices.Search.QueryTimeInMillis-prevNode.Indices.Search.QueryTimeInMillis))

				row.IndexingRate = clampRate(idxOpsDelta / elapsedSec)
				row.SearchRate = clampRate(srchOpsDelta / elapsedSec)
				row.IndexLatency = clampLatency(safeDivide(idxTimeDelta, idxOpsDelta))
				row.SearchLatency = clampLatency(safeDivide(srchTimeDelta, srchOpsDelta))
			} else {
				// Node not in prev (newly appeared) — delta cannot be computed.
				row.IndexingRate = model.MetricNotAvailable
				row.SearchRate = model.MetricNotAvailable
				row.IndexLatency = model.MetricNotAvailable
				row.SearchLatency = model.MetricNotAvailable
			}
		} else {
			row.IndexingRate = model.MetricNotAvailable
			row.SearchRate = model.MetricNotAvailable
			row.IndexLatency = model.MetricNotAvailable
			row.SearchLatency = model.MetricNotAvailable
		}

		rows = append(rows, row)
	}

	// Sort by node name for stable display order across polls.
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})

	return rows
}

// Sanity bounds ported from performanceTracker.ts lines 96-100.
const (
	minTimeDiffSeconds = 1.0
	maxRatePerSec      = 50_000_000.0
	maxLatencyMs       = 300_000.0
)

// clampRate returns 0 if r exceeds maxRatePerSec (counter wrap / bad data),
// otherwise returns r unchanged.
func clampRate(r float64) float64 {
	if r > maxRatePerSec {
		return 0
	}
	return r
}

// clampLatency caps l at maxLatencyMs.
func clampLatency(l float64) float64 {
	if l > maxLatencyMs {
		return maxLatencyMs
	}
	return l
}

// safeDivide returns a/b, or 0 when b is zero.
func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// maxFloat64 returns the larger of a and b.
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// CalcClusterMetrics computes cluster-level throughput and latency from the delta
// between two consecutive snapshots. Aggregates indexing ops from primaries and
// search ops from totals across all index stats, per the spec.
//
// Returns MetricNotAvailable for all rate/latency fields when:
//   - prev or curr is nil (first snapshot, no baseline)
//   - elapsed < minTimeDiffSeconds (interval too short, data unreliable)
func CalcClusterMetrics(prev, curr *model.Snapshot, elapsed time.Duration) model.PerformanceMetrics {
	if prev == nil || curr == nil || elapsed.Seconds() < minTimeDiffSeconds {
		return model.PerformanceMetrics{
			IndexingRate:  model.MetricNotAvailable,
			SearchRate:    model.MetricNotAvailable,
			IndexLatency:  model.MetricNotAvailable,
			SearchLatency: model.MetricNotAvailable,
		}
	}

	var (
		prevIndexOps   int64
		prevIndexTime  int64
		prevSearchOps  int64
		prevSearchTime int64
		currIndexOps   int64
		currIndexTime  int64
		currSearchOps  int64
		currSearchTime int64
	)

	// Aggregate indexing (primaries) and search (total) across indices present in
	// both curr and prev. Indices absent from prev are skipped entirely so their
	// cumulative counters are not treated as interval deltas, preventing spurious
	// rate/latency spikes when a new index appears between polls.
	for name, entry := range curr.IndexStats.Indices {
		prevEntry, ok := prev.IndexStats.Indices[name]
		if !ok {
			continue // new index — no baseline, cannot compute a valid delta
		}

		idxShard := entry.Primaries
		if idxShard == nil {
			idxShard = entry.Total
		}
		if idxShard != nil && idxShard.Indexing != nil {
			currIndexOps += idxShard.Indexing.IndexTotal
			currIndexTime += idxShard.Indexing.IndexTimeInMillis
		}
		srchShard := entry.Total
		if srchShard == nil {
			srchShard = entry.Primaries
		}
		if srchShard != nil && srchShard.Search != nil {
			currSearchOps += srchShard.Search.QueryTotal
			currSearchTime += srchShard.Search.QueryTimeInMillis
		}

		pidxShard := prevEntry.Primaries
		if pidxShard == nil {
			pidxShard = prevEntry.Total
		}
		if pidxShard != nil && pidxShard.Indexing != nil {
			prevIndexOps += pidxShard.Indexing.IndexTotal
			prevIndexTime += pidxShard.Indexing.IndexTimeInMillis
		}
		psrchShard := prevEntry.Total
		if psrchShard == nil {
			psrchShard = prevEntry.Primaries
		}
		if psrchShard != nil && psrchShard.Search != nil {
			prevSearchOps += psrchShard.Search.QueryTotal
			prevSearchTime += psrchShard.Search.QueryTimeInMillis
		}
	}

	elapsedSec := elapsed.Seconds()

	// Counter reset protection: clamp negative deltas to zero.
	indexOpsDelta := maxFloat64(0, float64(currIndexOps-prevIndexOps))
	searchOpsDelta := maxFloat64(0, float64(currSearchOps-prevSearchOps))
	indexTimeDelta := maxFloat64(0, float64(currIndexTime-prevIndexTime))
	searchTimeDelta := maxFloat64(0, float64(currSearchTime-prevSearchTime))

	indexingRate := clampRate(indexOpsDelta / elapsedSec)
	searchRate := clampRate(searchOpsDelta / elapsedSec)
	// Latency: deltaTime / deltaOps (interval-based, not cumulative).
	indexLatency := clampLatency(safeDivide(indexTimeDelta, indexOpsDelta))
	searchLatency := clampLatency(safeDivide(searchTimeDelta, searchOpsDelta))

	return model.PerformanceMetrics{
		IndexingRate:  indexingRate,
		SearchRate:    searchRate,
		IndexLatency:  indexLatency,
		SearchLatency: searchLatency,
	}
}

// CalcClusterResources aggregates OS, JVM, and filesystem metrics across all nodes
// in the snapshot. Ported from App.tsx lines 193-240.
//
// CPU and JVM averaging skips nodes that report 0 (offline or unsampled) to avoid
// dragging the average down. Storage is always summed across all nodes.
func CalcClusterResources(snap *model.Snapshot) model.ClusterResources {
	if snap == nil {
		return model.ClusterResources{}
	}

	var cpuSum float64
	var cpuCount int
	var jvmSum float64
	var jvmCount int
	var storageTotalBytes int64
	var storageUsedBytes int64

	for _, node := range snap.NodeStats.Nodes {
		// CPU: use os.cpu.percent, skip zeros.
		if node.OS != nil {
			cpu := float64(node.OS.CPU.Percent)
			if cpu > 0 {
				cpuSum += cpu
				cpuCount++
			}
		}

		// JVM heap: per-node used/max * 100, skip nodes with no heap (max=0).
		if node.JVM != nil {
			heapMax := node.JVM.Mem.HeapMaxInBytes
			heapUsed := node.JVM.Mem.HeapUsedInBytes
			if heapMax > 0 {
				heapPercent := float64(heapUsed) / float64(heapMax) * 100
				jvmSum += heapPercent
				jvmCount++
			}
		}

		// Storage: sum total and used across all nodes.
		if node.FS != nil {
			total := node.FS.Total.TotalInBytes
			available := node.FS.Total.AvailableInBytes
			storageTotalBytes += total
			used := total - available
			if used < 0 {
				used = 0
			}
			storageUsedBytes += used
		}
	}

	storagePercent := safeDivide(float64(storageUsedBytes), float64(storageTotalBytes)) * 100

	return model.ClusterResources{
		AvgCPUPercent:     safeDivide(cpuSum, float64(cpuCount)),
		AvgJVMHeapPercent: safeDivide(jvmSum, float64(jvmCount)),
		StorageUsedBytes:  storageUsedBytes,
		StorageTotalBytes: storageTotalBytes,
		StoragePercent:    storagePercent,
	}
}

// CalcIndexRows computes per-index throughput, latency, and size metrics from two
// consecutive snapshots. Ported from IndexTable.tsx lines 66-131.
//
// Critical primaries-vs-total rule (IndexTable.tsx lines 73-76):
//   - Indexing ops/time: use primaries (fallback to total if primaries nil)
//   - Search ops/time:   use total     (fallback to primaries if total nil)
func CalcIndexRows(prev, curr *model.Snapshot, elapsed time.Duration) []model.IndexRow {
	if curr == nil {
		return nil
	}

	// Build prev index stats lookup (nil safe).
	var prevStats map[string]client.IndexStatEntry
	if prev != nil {
		prevStats = prev.IndexStats.Indices
	}

	elapsedSec := elapsed.Seconds()
	enoughTime := prev != nil && elapsedSec >= minTimeDiffSeconds

	rows := make([]model.IndexRow, 0, len(curr.Indices))
	for _, info := range curr.Indices {
		name := info.Index

		// Parse string fields from _cat/indices response.
		// ES returns "-" for unavailable fields (e.g. some system indices);
		// treat any non-numeric value as 0 rather than silently discarding the error.
		pri := 0
		if v, err := strconv.Atoi(info.Pri); err == nil {
			pri = v
		}
		rep := 0
		if v, err := strconv.Atoi(info.Rep); err == nil {
			rep = v
		}
		docCount := int64(0)
		if v, err := strconv.ParseInt(info.DocsCount, 10, 64); err == nil {
			docCount = v
		}
		totalShards := pri * (1 + rep)

		// Size from _stats shard data.
		var totalSizeBytes int64
		var primarySizeBytes int64
		if entry, ok := curr.IndexStats.Indices[name]; ok {
			if entry.Total != nil && entry.Total.Store != nil {
				totalSizeBytes = entry.Total.Store.SizeInBytes
			}
			if entry.Primaries != nil && entry.Primaries.Store != nil {
				primarySizeBytes = entry.Primaries.Store.SizeInBytes
			}
		}

		priCount := int64(pri)
		if priCount < 1 {
			priCount = 1
		}
		avgShardSize := primarySizeBytes / priCount

		row := model.IndexRow{
			Name:           name,
			PrimaryShards:  pri,
			TotalShards:    totalShards,
			TotalSizeBytes: totalSizeBytes,
			AvgShardSize:   avgShardSize,
			DocCount:       docCount,
		}

		if enoughTime {
			var currIdxOps, currIdxTime int64
			var prevIdxOps, prevIdxTime int64
			var currSrchOps, currSrchTime int64
			var prevSrchOps, prevSrchTime int64

			if entry, ok := curr.IndexStats.Indices[name]; ok {
				// Indexing: primaries preferred, fallback to total.
				idxShard := entry.Primaries
				if idxShard == nil {
					idxShard = entry.Total
				}
				if idxShard != nil && idxShard.Indexing != nil {
					currIdxOps = idxShard.Indexing.IndexTotal
					currIdxTime = idxShard.Indexing.IndexTimeInMillis
				}

				// Search: total preferred, fallback to primaries.
				srchShard := entry.Total
				if srchShard == nil {
					srchShard = entry.Primaries
				}
				if srchShard != nil && srchShard.Search != nil {
					currSrchOps = srchShard.Search.QueryTotal
					currSrchTime = srchShard.Search.QueryTimeInMillis
				}
			}

			if prevStats != nil {
				prevEntry, prevOk := prevStats[name]
				if !prevOk {
					// New index: no previous baseline, cannot compute a valid delta.
					row.IndexingRate = model.MetricNotAvailable
					row.SearchRate = model.MetricNotAvailable
					row.IndexLatency = model.MetricNotAvailable
					row.SearchLatency = model.MetricNotAvailable
					rows = append(rows, row)
					continue
				}

				idxShard := prevEntry.Primaries
				if idxShard == nil {
					idxShard = prevEntry.Total
				}
				if idxShard != nil && idxShard.Indexing != nil {
					prevIdxOps = idxShard.Indexing.IndexTotal
					prevIdxTime = idxShard.Indexing.IndexTimeInMillis
				}

				srchShard := prevEntry.Total
				if srchShard == nil {
					srchShard = prevEntry.Primaries
				}
				if srchShard != nil && srchShard.Search != nil {
					prevSrchOps = srchShard.Search.QueryTotal
					prevSrchTime = srchShard.Search.QueryTimeInMillis
				}
			} else {
				// prev exists but its IndexStats.Indices is nil (e.g. ES returned
				// "indices": null): no valid previous baseline, mark unavailable.
				row.IndexingRate = model.MetricNotAvailable
				row.SearchRate = model.MetricNotAvailable
				row.IndexLatency = model.MetricNotAvailable
				row.SearchLatency = model.MetricNotAvailable
				rows = append(rows, row)
				continue
			}

			idxOpsDelta := maxFloat64(0, float64(currIdxOps-prevIdxOps))
			idxTimeDelta := maxFloat64(0, float64(currIdxTime-prevIdxTime))
			srchOpsDelta := maxFloat64(0, float64(currSrchOps-prevSrchOps))
			srchTimeDelta := maxFloat64(0, float64(currSrchTime-prevSrchTime))

			row.IndexingRate = clampRate(idxOpsDelta / elapsedSec)
			row.SearchRate = clampRate(srchOpsDelta / elapsedSec)
			row.IndexLatency = clampLatency(safeDivide(idxTimeDelta, idxOpsDelta))
			row.SearchLatency = clampLatency(safeDivide(srchTimeDelta, srchOpsDelta))
		} else {
			row.IndexingRate = model.MetricNotAvailable
			row.SearchRate = model.MetricNotAvailable
			row.IndexLatency = model.MetricNotAvailable
			row.SearchLatency = model.MetricNotAvailable
		}

		rows = append(rows, row)
	}

	return rows
}
