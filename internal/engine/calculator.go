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
