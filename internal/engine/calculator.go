package engine

import (
	"time"

	"github.com/dm/epm-go/internal/model"
)

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
// between two consecutive snapshots. It aggregates indexing and search counters
// across all nodes in NodeStats.
//
// Returns zero PerformanceMetrics when:
//   - prev is nil (first snapshot, no baseline)
//   - elapsed < minTimeDiffSeconds (interval too short, data unreliable)
func CalcClusterMetrics(prev, curr *model.Snapshot, elapsed time.Duration) model.PerformanceMetrics {
	if prev == nil || elapsed.Seconds() < minTimeDiffSeconds {
		return model.PerformanceMetrics{}
	}

	var (
		prevIndexOps  int64
		prevIndexTime int64
		prevSearchOps int64
		prevSearchTime int64
		currIndexOps  int64
		currIndexTime int64
		currSearchOps int64
		currSearchTime int64
	)

	for _, node := range prev.NodeStats.Nodes {
		if node.Indices == nil {
			continue
		}
		prevIndexOps += node.Indices.Indexing.IndexTotal
		prevIndexTime += node.Indices.Indexing.IndexTimeInMillis
		prevSearchOps += node.Indices.Search.QueryTotal
		prevSearchTime += node.Indices.Search.QueryTimeInMillis
	}

	for _, node := range curr.NodeStats.Nodes {
		if node.Indices == nil {
			continue
		}
		currIndexOps += node.Indices.Indexing.IndexTotal
		currIndexTime += node.Indices.Indexing.IndexTimeInMillis
		currSearchOps += node.Indices.Search.QueryTotal
		currSearchTime += node.Indices.Search.QueryTimeInMillis
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

		// JVM heap: per-node used/max * 100, skip zeros.
		if node.JVM != nil {
			heapMax := node.JVM.Mem.HeapMaxInBytes
			heapUsed := node.JVM.Mem.HeapUsedInBytes
			if heapMax > 0 {
				heapPercent := float64(heapUsed) / float64(heapMax) * 100
				if heapPercent > 0 {
					jvmSum += heapPercent
					jvmCount++
				}
			}
		}

		// Storage: sum total and used across all nodes.
		if node.FS != nil {
			total := node.FS.Total.TotalInBytes
			available := node.FS.Total.AvailableInBytes
			storageTotalBytes += total
			storageUsedBytes += total - available
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
