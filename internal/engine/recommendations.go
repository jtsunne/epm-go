package engine

import (
	"fmt"
	"strings"

	"github.com/jtsunne/epm-go/internal/model"
)

const (
	oneGiBInt64 = int64(1 << 30) // 1 GiB in bytes
	oneMiBInt64 = int64(1 << 20) // 1 MiB in bytes
)

// CalcRecommendations generates actionable recommendations for the cluster
// based on the current snapshot, resources, and computed rows.
// Returns an empty (non-nil) slice when snap is nil or data is unavailable.
func CalcRecommendations(
	snap *model.Snapshot,
	resources model.ClusterResources,
	nodeRows []model.NodeRow,
	indexRows []model.IndexRow,
) []model.Recommendation {
	result := []model.Recommendation{}
	if snap == nil {
		return result
	}

	// Cluster status.
	switch snap.Health.Status {
	case "red":
		result = append(result, model.Recommendation{
			Severity: model.SeverityCritical,
			Category: model.CategoryShardHealth,
			Title:    "Cluster status RED",
			Detail:   "Cluster is in RED status — one or more primary shards are unavailable. Data may be missing. Investigate unassigned shards immediately.",
		})
	case "yellow":
		result = append(result, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryShardHealth,
			Title:    "Cluster status YELLOW",
			Detail:   "Cluster is in YELLOW status — some replica shards are unassigned. Data is available but redundancy is reduced.",
		})
	}

	// Unassigned shards.
	if snap.Health.UnassignedShards > 0 {
		result = append(result, model.Recommendation{
			Severity: model.SeverityCritical,
			Category: model.CategoryShardHealth,
			Title:    "Unassigned shards detected",
			Detail:   fmt.Sprintf("%d unassigned shards detected. Check node availability and disk space.", snap.Health.UnassignedShards),
		})
	}

	// CPU pressure.
	switch {
	case resources.AvgCPUPercent > 90:
		result = append(result, model.Recommendation{
			Severity: model.SeverityCritical,
			Category: model.CategoryResourcePressure,
			Title:    "Critical CPU pressure",
			Detail:   fmt.Sprintf("Average cluster CPU at %.0f%%. Critical load risks query timeouts and node instability. Add data nodes or reduce indexing throughput immediately.", resources.AvgCPUPercent),
		})
	case resources.AvgCPUPercent > 80:
		result = append(result, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryResourcePressure,
			Title:    "High CPU usage",
			Detail:   fmt.Sprintf("Average cluster CPU at %.0f%%. Sustained high CPU can degrade query performance. Consider adding data nodes or reducing indexing load.", resources.AvgCPUPercent),
		})
	}

	// JVM heap pressure.
	totalHeapGB := float64(resources.TotalHeapMaxBytes) / float64(oneGiBInt64)
	switch {
	case resources.AvgJVMHeapPercent > 85:
		result = append(result, model.Recommendation{
			Severity: model.SeverityCritical,
			Category: model.CategoryResourcePressure,
			Title:    "Critical JVM heap pressure",
			Detail:   fmt.Sprintf("Average JVM heap at %.0f%% (%.1f GB total heap). At this level GC pauses are severe and OOM risk is high. Increase node heap (max 32 GB) or add nodes urgently.", resources.AvgJVMHeapPercent, totalHeapGB),
		})
	case resources.AvgJVMHeapPercent > 75:
		result = append(result, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryResourcePressure,
			Title:    "High JVM heap usage",
			Detail:   fmt.Sprintf("Average JVM heap at %.0f%% (%.1f GB total heap). Elevated heap increases GC frequency. Consider increasing node heap (max 32 GB) or adding nodes.", resources.AvgJVMHeapPercent, totalHeapGB),
		})
	}

	// Storage pressure.
	switch {
	case resources.StoragePercent > 90:
		result = append(result, model.Recommendation{
			Severity: model.SeverityCritical,
			Category: model.CategoryResourcePressure,
			Title:    "Critical storage usage",
			Detail:   fmt.Sprintf("Cluster storage at %.0f%% capacity. Immediate action required — ES will reject writes at 95%%. Delete old indices or add storage.", resources.StoragePercent),
		})
	case resources.StoragePercent > 80:
		result = append(result, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryResourcePressure,
			Title:    "High storage usage",
			Detail:   fmt.Sprintf("Cluster storage at %.0f%% capacity. Plan capacity expansion or enable ILM to delete old data.", resources.StoragePercent),
		})
	}

	// Shard-to-heap ratio — resource-aware dynamic threshold.
	if resources.TotalHeapMaxBytes > 0 {
		heapGB := totalHeapGB
		activeShards := snap.Health.ActiveShards
		shardPerGB := float64(activeShards) / heapGB
		maxIdeal := int(20 * heapGB)
		switch {
		case shardPerGB > 40:
			result = append(result, model.Recommendation{
				Severity: model.SeverityCritical,
				Category: model.CategoryShardHealth,
				Title:    "Too many shards per GB heap (critical)",
				Detail:   fmt.Sprintf("Cluster has %d shards across %.1f GB heap (%.0f/GB). Ideal max: %d shards. Remove unused indices or reduce primary shard count.", activeShards, heapGB, shardPerGB, maxIdeal),
			})
		case shardPerGB > 20:
			result = append(result, model.Recommendation{
				Severity: model.SeverityWarning,
				Category: model.CategoryShardHealth,
				Title:    "Too many shards per GB heap",
				Detail:   fmt.Sprintf("Cluster has %d shards across %.1f GB heap (%.0f/GB). Ideal max: %d shards. Remove unused indices or reduce primary shard count.", activeShards, heapGB, shardPerGB, maxIdeal),
			})
		}
	}

	// Aggregate index-level data.
	var totalIndexSizeBytes int64
	var totalShardCount int
	var zeroReplicaCount int
	for _, idx := range indexRows {
		totalIndexSizeBytes += idx.TotalSizeBytes
		totalShardCount += idx.TotalShards
		// Non-system index with explicitly 0 replicas: rep parsed successfully and
		// TotalShards == PrimaryShards. Skip indices where rep was unavailable ("-")
		// to avoid false positives on closed or partially-initialised indices.
		if !strings.HasPrefix(idx.Name, ".") && idx.RepKnown && idx.PrimaryShards > 0 && idx.TotalShards == idx.PrimaryShards {
			zeroReplicaCount++
		}
	}

	// Zero-replica indices.
	if zeroReplicaCount > 0 {
		result = append(result, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryIndexConfig,
			Title:    "Indices without replicas",
			Detail:   fmt.Sprintf("%d indices have no replicas. A single node failure will cause data loss.", zeroReplicaCount),
		})
	}

	// Average shard size checks.
	if totalShardCount > 0 {
		avgShardBytes := totalIndexSizeBytes / int64(totalShardCount)
		avgShardGB := float64(avgShardBytes) / float64(oneGiBInt64)
		avgShardMB := float64(avgShardBytes) / float64(oneMiBInt64)

		if avgShardGB > 50 {
			result = append(result, model.Recommendation{
				Severity: model.SeverityWarning,
				Category: model.CategoryIndexConfig,
				Title:    "Oversized shards",
				Detail:   fmt.Sprintf("Average shard size %.1f GB exceeds 50 GB. Large shards slow recovery and rebalancing. Split high-volume indices.", avgShardGB),
			})
		}

		dataNodeCount := countDataNodes(nodeRows)
		if dataNodeCount < 1 {
			dataNodeCount = 1
		}
		if avgShardGB < 1.0 && totalShardCount > 10*dataNodeCount {
			result = append(result, model.Recommendation{
				Severity: model.SeverityWarning,
				Category: model.CategoryIndexConfig,
				Title:    "Over-sharding detected",
				Detail:   fmt.Sprintf("Average shard size %.0f MB is very small across %d shards. Over-sharding wastes heap. Merge small indices or increase ILM rollover size.", avgShardMB, totalShardCount),
			})
		}
	}

	// Data-to-heap ratio.
	if resources.TotalHeapMaxBytes > 0 && totalIndexSizeBytes > 0 {
		heapGB := totalHeapGB
		dataGB := float64(totalIndexSizeBytes) / float64(oneGiBInt64)
		ratio := float64(totalIndexSizeBytes) / float64(resources.TotalHeapMaxBytes)
		if ratio > 30 {
			result = append(result, model.Recommendation{
				Severity: model.SeverityWarning,
				Category: model.CategoryResourcePressure,
				Title:    "High data-to-heap ratio",
				Detail:   fmt.Sprintf("Index data (%.1f GB) is %.0f× total heap (%.1f GB). Elastic recommends ≤30× for search workloads. Add data nodes or reduce index retention.", dataGB, ratio, heapGB),
			})
		}
	}

	// Single data node SPOF.
	if countDataNodes(nodeRows) == 1 {
		result = append(result, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryShardHealth,
			Title:    "Single data node — no redundancy",
			Detail:   "Only 1 data node — no replica can be assigned. Add a second data node for high availability.",
		})
	}

	// Per-node heap hotspot.
	result = append(result, heapHotspotRecs(nodeRows)...)

	return result
}

// heapHotspotRecs returns a warning recommendation when heap utilisation spread
// across nodes exceeds 30 percentage points, or nil when the cluster is healthy.
func heapHotspotRecs(nodeRows []model.NodeRow) []model.Recommendation {
	if len(nodeRows) < 2 {
		return nil
	}
	var minUtil, maxUtil float64
	first := true
	for _, n := range nodeRows {
		if n.HeapMaxBytes <= 0 {
			continue
		}
		util := float64(n.HeapUsedBytes) / float64(n.HeapMaxBytes)
		if first {
			minUtil, maxUtil = util, util
			first = false
		} else {
			if util < minUtil {
				minUtil = util
			}
			if util > maxUtil {
				maxUtil = util
			}
		}
	}
	if !first && (maxUtil-minUtil) > 0.30 {
		return []model.Recommendation{{
			Severity: model.SeverityWarning,
			Category: model.CategoryHotspot,
			Title:    "Uneven heap utilization across nodes",
			Detail: fmt.Sprintf(
				"Uneven heap utilization across nodes (high: %.0f%%, low: %.0f%%). Rebalance shards with `_cluster/reroute` or enable `cluster.routing.rebalance.enable`.",
				maxUtil*100, minUtil*100,
			),
		}}
	}
	return nil
}

// countDataNodes counts nodes whose role string contains 'd' (data role).
func countDataNodes(nodeRows []model.NodeRow) int {
	count := 0
	for _, n := range nodeRows {
		if strings.ContainsRune(n.Role, 'd') {
			count++
		}
	}
	return count
}
