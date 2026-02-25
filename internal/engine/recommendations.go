package engine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jtsunne/epm-go/internal/model"
)

const (
	oneGiBInt64        = int64(1 << 30) // 1 GiB in bytes
	oneMiBInt64        = int64(1 << 20) // 1 MiB in bytes
	rollupThresholdMiB = int64(100)     // daily→monthly if avg primary < 100 MiB; else →weekly
)

// Package-level compiled regexes for date-patterned index detection.
// Priority order: daily checked first to avoid misclassifying YYYY.MM.DD as monthly.
var (
	reIndexDaily   = regexp.MustCompile(`^(.+)[.\-_](\d{4})[.\-](\d{2})[.\-](\d{2})$`)
	reIndexWeekly  = regexp.MustCompile(`^(.+)[.\-_](\d{4})[.\-][Ww](\d{1,2})$`)
	reIndexMonthly = regexp.MustCompile(`^(.+)[.\-_](\d{4})[.\-](\d{2})$`)
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

	// Cluster status — unassigned shard count is included in the detail to
	// avoid a separate duplicate recommendation for the same root cause.
	// RED is caused by unassigned primary shards; YELLOW by unassigned replicas.
	switch snap.Health.Status {
	case "red":
		detail := "Cluster is in RED status — one or more primary shards are unavailable. Data may be missing. Investigate unassigned shards immediately."
		if snap.Health.UnassignedShards > 0 {
			detail = fmt.Sprintf("Cluster is in RED status — %d unassigned shard(s), including primary shards. Data may be missing. Check node availability and disk space immediately.", snap.Health.UnassignedShards)
		}
		result = append(result, model.Recommendation{
			Severity: model.SeverityCritical,
			Category: model.CategoryShardHealth,
			Title:    "Cluster status RED",
			Detail:   detail,
		})
	case "yellow":
		detail := "Cluster is in YELLOW status — some replica shards are unassigned. Data is available but redundancy is reduced."
		if snap.Health.UnassignedShards > 0 {
			detail = fmt.Sprintf("Cluster is in YELLOW status — %d unassigned replica shard(s). Data is available but redundancy is reduced. Check node capacity.", snap.Health.UnassignedShards)
		}
		result = append(result, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryShardHealth,
			Title:    "Cluster status YELLOW",
			Detail:   detail,
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
		activeShards := snap.Health.ActiveShards
		shardPerGB := float64(activeShards) / totalHeapGB
		maxIdeal := int(20 * totalHeapGB)
		switch {
		case shardPerGB > 40:
			result = append(result, model.Recommendation{
				Severity: model.SeverityCritical,
				Category: model.CategoryShardHealth,
				Title:    "Too many shards per GB heap (critical)",
				Detail:   fmt.Sprintf("Cluster has %d shards across %.1f GB heap (%.0f/GB). Ideal max: %d shards. Remove unused indices or reduce primary shard count.", activeShards, totalHeapGB, shardPerGB, maxIdeal),
			})
		case shardPerGB > 20:
			result = append(result, model.Recommendation{
				Severity: model.SeverityWarning,
				Category: model.CategoryShardHealth,
				Title:    "Too many shards per GB heap",
				Detail:   fmt.Sprintf("Cluster has %d shards across %.1f GB heap (%.0f/GB). Ideal max: %d shards. Remove unused indices or reduce primary shard count.", activeShards, totalHeapGB, shardPerGB, maxIdeal),
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
		detail := fmt.Sprintf("%d indices have no replicas. A single node failure will cause data loss.", zeroReplicaCount)
		if zeroReplicaCount == 1 {
			detail = "1 index has no replicas. A single node failure will cause data loss."
		}
		result = append(result, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryIndexConfig,
			Title:    "Indices without replicas",
			Detail:   detail,
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
		dataGB := float64(totalIndexSizeBytes) / float64(oneGiBInt64)
		ratio := float64(totalIndexSizeBytes) / float64(resources.TotalHeapMaxBytes)
		if ratio > 30 {
			result = append(result, model.Recommendation{
				Severity: model.SeverityWarning,
				Category: model.CategoryResourcePressure,
				Title:    "High data-to-heap ratio",
				Detail:   fmt.Sprintf("Index data (%.1f GB) is %.0f× total heap (%.1f GB). Elastic recommends ≤30× for search workloads. Add data nodes or reduce index retention.", dataGB, ratio, totalHeapGB),
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

	// Index lifecycle: date-rollup consolidation suggestions.
	rollupRecs, savedIdx, savedShards := dateRollupRecs(indexRows)
	result = append(result, rollupRecs...)

	// Index lifecycle: empty index detection.
	result = append(result, emptyIndexRecs(indexRows)...)

	// Cluster-level impact summary for rollup recommendations.
	if savedShards > 0 && resources.TotalHeapMaxBytes > 0 {
		activeShards := snap.Health.ActiveShards
		currentRatio := float64(activeShards) / totalHeapGB
		estimatedShards := activeShards - savedShards
		if estimatedShards < 0 {
			estimatedShards = 0
		}
		estimatedRatio := float64(estimatedShards) / totalHeapGB
		result = append(result, model.Recommendation{
			Severity: model.SeverityNormal,
			Category: model.CategoryIndexLifecycle,
			Title:    "Rollup impact summary",
			Detail: fmt.Sprintf(
				"Applying all rollup suggestions would eliminate ~%d indices (~%d shards). "+
					"Shard/GB heap: current %.1f → estimated %.1f after rollup.",
				savedIdx, savedShards, currentRatio, estimatedRatio,
			),
		})
	}

	return result
}

// dateRollupGroupKey identifies a group of date-patterned indices.
type dateRollupGroupKey struct {
	granularity string // "daily", "weekly", or "monthly"
	base        string // base index name without the date suffix
}

// dateRollupRecs analyses date-patterned indices and emits consolidation
// recommendations when enough indices exist to justify a rollup. Returns the
// recommendations plus aggregate savedIndices and savedShards counts for use in
// the cluster-level impact summary.
func dateRollupRecs(indexRows []model.IndexRow) (recs []model.Recommendation, savedIndices int, savedShards int) {
	// Group indices by (granularity, base).
	groups := make(map[dateRollupGroupKey][]model.IndexRow)

	for _, idx := range indexRows {
		// Skip system indices.
		if strings.HasPrefix(idx.Name, ".") {
			continue
		}
		var key dateRollupGroupKey
		if m := reIndexDaily.FindStringSubmatch(idx.Name); m != nil {
			key = dateRollupGroupKey{granularity: "daily", base: m[1]}
		} else if m := reIndexWeekly.FindStringSubmatch(idx.Name); m != nil {
			key = dateRollupGroupKey{granularity: "weekly", base: m[1]}
		} else if m := reIndexMonthly.FindStringSubmatch(idx.Name); m != nil {
			key = dateRollupGroupKey{granularity: "monthly", base: m[1]}
		} else {
			continue
		}
		groups[key] = append(groups[key], idx)
	}

	// Sort keys for deterministic output order.
	keys := make([]dateRollupGroupKey, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].granularity != keys[j].granularity {
			return keys[i].granularity < keys[j].granularity
		}
		return keys[i].base < keys[j].base
	})

	for _, key := range keys {
		group := groups[key]
		n := len(group)

		// Determine minimum count threshold, consolidation target, and period size.
		var minCount, periodSize int
		var target string
		switch key.granularity {
		case "daily":
			minCount = 7
			if n < minCount {
				continue
			}
			// Size-aware: small daily indices skip the weekly step and go straight to monthly.
			var sumPri int64
			for _, idx := range group {
				sumPri += idx.PriSizeBytes
			}
			avgPriSize := sumPri / int64(n)
			if avgPriSize < rollupThresholdMiB*oneMiBInt64 {
				target = "monthly"
				periodSize = 30
			} else {
				target = "weekly"
				periodSize = 7
			}
		case "weekly":
			minCount = 4
			if n < minCount {
				continue
			}
			target = "monthly"
			periodSize = 4
		case "monthly":
			minCount = 12
			if n < minCount {
				continue
			}
			target = "yearly"
			periodSize = 12
		default:
			continue
		}

		// Compute impact metrics.
		var sumPriBytes int64
		var sumTotalShards int
		for _, idx := range group {
			sumPriBytes += idx.PriSizeBytes
			sumTotalShards += idx.TotalShards
		}

		avgPriMiB := float64(sumPriBytes) / float64(n) / float64(oneMiBInt64)
		totalPriGiB := float64(sumPriBytes) / float64(oneGiBInt64)

		// M = ceil(N / periodSize) using integer arithmetic.
		m := (n + periodSize - 1) / periodSize
		if m < 1 {
			m = 1
		}

		avgShardDensity := sumTotalShards / n // integer division — proportional estimate
		grpSavedIndices := n - m
		grpSavedShards := grpSavedIndices * avgShardDensity

		var sizePerConsolidatedMiB float64
		if m > 0 {
			sizePerConsolidatedMiB = float64(sumPriBytes) / float64(m) / float64(oneMiBInt64)
		}

		savedIndices += grpSavedIndices
		savedShards += grpSavedShards

		detail := fmt.Sprintf(
			"Current: %d indices, avg %.0f MiB primary/index, total %.2f GiB primary.\n"+
				"After consolidation to %s: ~%d %s indices, ~%d fewer shards, ~%.0f MiB per consolidated index.",
			n, avgPriMiB, totalPriGiB,
			target, m, target, grpSavedShards, sizePerConsolidatedMiB,
		)

		recs = append(recs, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryIndexLifecycle,
			Title:    fmt.Sprintf("Consolidate %s '%s' indices → %s", key.granularity, key.base, target),
			Detail:   detail,
		})
	}

	return recs, savedIndices, savedShards
}

// emptyIndexRecs returns a warning recommendation when three or more non-system
// indices have zero documents and zero storage — likely stale or forgotten indices
// that can be safely deleted.
func emptyIndexRecs(indexRows []model.IndexRow) []model.Recommendation {
	var names []string
	for _, idx := range indexRows {
		if strings.HasPrefix(idx.Name, ".") {
			continue
		}
		if idx.DocCountKnown && idx.DocCount == 0 && idx.TotalSizeBytes == 0 {
			names = append(names, idx.Name)
		}
	}
	if len(names) < 3 {
		return nil
	}
	const maxListed = 5
	var listed string
	if len(names) <= maxListed {
		listed = strings.Join(names, ", ")
	} else {
		listed = strings.Join(names[:maxListed], ", ") + fmt.Sprintf(", ... and %d more", len(names)-maxListed)
	}
	return []model.Recommendation{{
		Severity: model.SeverityWarning,
		Category: model.CategoryIndexLifecycle,
		Title:    fmt.Sprintf("%d empty indices found", len(names)),
		Detail:   fmt.Sprintf("Non-system indices with 0 docs and 0 storage are deletion candidates: %s", listed),
	}}
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

// countDataNodes counts nodes whose role string contains any data role abbreviation.
// 'd' = data (generic), 'h' = data_hot, 'w' = data_warm, 'c' = data_cold,
// 'f' = data_frozen, 's' = data_content (ES 8.x+ tiered roles).
func countDataNodes(nodeRows []model.NodeRow) int {
	count := 0
	for _, n := range nodeRows {
		if strings.ContainsAny(n.Role, "dhwcfs") {
			count++
		}
	}
	return count
}
