# Analytics Screen: Resource-Aware Cluster Recommendations

## Overview

Add a full-screen Analytics view (key `a`) that displays actionable recommendations tailored to the cluster's actual hardware resources. Recommendations compute dynamic thresholds from heap size — e.g., a 4GB-RAM node gets max ~40 shards (20/GB heap), a 64GB-RAM node gets max ~640 shards. Press `a` or `esc` to return to the dashboard.

## Context

- Files involved:
  - `internal/client/endpoints.go` — add `unassigned_shards,number_of_pending_tasks` to health filter_path
  - `internal/client/types.go` — add `UnassignedShards`, `NumberOfPendingTasks` to `ClusterHealth`
  - `internal/client/client_test.go` — update health fixture JSON
  - `internal/model/metrics.go` — add `TotalHeapMaxBytes int64` to `ClusterResources`; add `HeapMaxBytes int64`, `HeapUsedBytes int64` to `NodeRow`
  - `internal/engine/calculator.go` — update `CalcClusterResources` to sum heap_max; update `CalcNodeRows` to populate heap fields
  - `internal/engine/calculator_test.go` — update existing tests
  - `internal/model/recommendation.go` — new: `Recommendation` type, category/severity constants
  - `internal/engine/recommendations.go` — new: `CalcRecommendations` function (resource-aware)
  - `internal/engine/recommendations_test.go` — new: table-driven tests
  - `internal/tui/messages.go` — add `Recommendations` field to `SnapshotMsg`
  - `internal/tui/app.go` — wire analytics mode, scroll, key handler, `fetchCmd` call
  - `internal/tui/keys.go` — add `Analytics` binding
  - `internal/tui/analytics.go` — new: `renderAnalytics(app)` full-screen renderer
- Related patterns: `CalcClusterResources` aggregation loop; `thresholds.go` severity; `renderMetrics`/`renderHeader` always-visible framing
- Dependencies: none new

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Propagate heap data and add unassigned/pending to health

**Files:**
- Modify: `internal/client/endpoints.go`
- Modify: `internal/client/types.go`
- Modify: `internal/model/metrics.go`
- Modify: `internal/engine/calculator.go`
- Modify: `internal/engine/calculator_test.go`
- Modify: `internal/client/client_test.go`

- [x] Add `unassigned_shards,number_of_pending_tasks` to `endpointClusterHealth` filter_path in `endpoints.go`
- [x] Add `UnassignedShards int` and `NumberOfPendingTasks int` to `ClusterHealth` struct in `types.go`
- [x] Add `TotalHeapMaxBytes int64` to `ClusterResources` in `model/metrics.go`
- [x] Add `HeapMaxBytes int64` and `HeapUsedBytes int64` to `NodeRow` in `model/metrics.go`
- [x] In `CalcClusterResources`: sum `node.JVM.Mem.HeapMaxInBytes` across all nodes into `TotalHeapMaxBytes`
- [x] In `CalcNodeRows`: populate `row.HeapMaxBytes` and `row.HeapUsedBytes` from `node.JVM.Mem` (zero when JVM is nil)
- [x] Update health fixture JSON in `client_test.go` to include `unassigned_shards` and `number_of_pending_tasks`
- [x] Update `CalcClusterResources` tests in `calculator_test.go` to assert `TotalHeapMaxBytes`
- [x] Update `CalcNodeRows` tests to assert `HeapMaxBytes` and `HeapUsedBytes`
- [x] Run `make test` — must pass before task 2

### Task 2: Add Recommendation model type

**Files:**
- Create: `internal/model/recommendation.go`

- [x] Define `RecommendationSeverity` type with constants: `SeverityNormal`, `SeverityWarning`, `SeverityCritical`
- [x] Define `RecommendationCategory` type with constants: `CategoryResourcePressure`, `CategoryShardHealth`, `CategoryIndexConfig`, `CategoryHotspot`
- [x] Define `Recommendation` struct: `Severity RecommendationSeverity`, `Category RecommendationCategory`, `Title string`, `Detail string`
- [x] Run `make test` — must pass before task 3

### Task 3: Resource-aware CalcRecommendations

**Files:**
- Create: `internal/engine/recommendations.go`
- Create: `internal/engine/recommendations_test.go`

Implement `CalcRecommendations(snap *model.Snapshot, metrics model.PerformanceMetrics, resources model.ClusterResources, nodeRows []model.NodeRow, indexRows []model.IndexRow) []model.Recommendation`.

Resource-aware rules — dynamic thresholds derived from `resources.TotalHeapMaxBytes`:

- [x] **Shard-to-heap ratio**: `ratio = activeShards / (TotalHeapMaxBytes / 1GiB)`; warning if > 20/GB, critical if > 40/GB. Detail: "Cluster has X shards across Y GB heap (Z/GB). Ideal max: [20*Y] shards. Remove unused indices or reduce primary shard count."
- [x] **Average shard size (oversized)**: `avgGB = sum(indexRow.TotalSizeBytes) / sum(indexRow.TotalShards) / 1GiB`; warning if avg > 50GB. Detail: "Average shard size X GB exceeds 50 GB. Large shards slow recovery and rebalancing. Split high-volume indices."
- [x] **Over-sharding (undersized)**: warning if avg < 1GB and total shard count > 10 per data node. Detail: "Average shard size X MB is very small across Z shards. Over-sharding wastes heap. Merge small indices or increase ILM rollover size."
- [x] **Data-to-heap ratio**: `ratio = sum(indexRow.TotalSizeBytes) / TotalHeapMaxBytes`; warning if > 30. Detail: "Index data (X GB) is Y× total heap (Z GB). Elastic recommends ≤30× for search workloads. Add data nodes or reduce index retention."
- [x] **CPU pressure**: warning if AvgCPUPercent > 80%, critical if > 90%. Detail: "Average cluster CPU at X%. Critical load risks query timeouts. Add data nodes or reduce indexing throughput."
- [x] **JVM heap pressure (with absolute values)**: warning if AvgJVMHeapPercent > 75%, critical if > 85%. Detail includes TotalHeapMaxBytes: "Average JVM heap at X% (Y GB total heap). At critical levels GC pauses impact latency. Increase node heap (max 32 GB) or add nodes."
- [x] **Storage pressure**: warning if StoragePercent > 80%, critical if > 90%
- [x] **Unassigned shards**: critical if UnassignedShards > 0. Detail: "X unassigned shards detected. Check node availability and disk space."
- [x] **Cluster status**: critical if status == "red", warning if "yellow"
- [x] **Single data node SPOF**: warning if data node count == 1. Detail: "Only 1 data node — no replica can be assigned. Add a second data node for high availability."
- [x] **Zero-replica indices** (non-system): warning if any `IndexRow.TotalShards == IndexRow.PrimaryShards`. Detail: "X indices have no replicas. A single node failure will cause data loss."
- [x] **Per-node heap hotspot**: warning if `max(HeapUsedBytes/HeapMaxBytes) - min(HeapUsedBytes/HeapMaxBytes) > 0.30` across nodes. Detail: "Uneven heap utilization across nodes (high: X%, low: Y%). Rebalance shards with `_cluster/reroute` or enable `cluster.routing.rebalance.enable`."
- [x] Return empty slice (not nil) when `snap == nil` or data is unavailable
- [x] Write table-driven tests covering: small-RAM scenario (4GB heap, 200 shards triggers critical), large-RAM scenario (64GB heap, 200 shards is fine), avg shard size checks, data-to-heap ratio, hotspot detection, zero-replica check
- [x] Run `make test` — must pass before task 4

### Task 4: Wire recommendations into app state and SnapshotMsg

**Files:**
- Modify: `internal/tui/messages.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/keys.go`

- [x] Add `Recommendations []model.Recommendation` to `SnapshotMsg` in `messages.go`
- [x] In `fetchCmd` in `app.go`, call `engine.CalcRecommendations(...)` and include result in `SnapshotMsg`
- [x] Add `recommendations []model.Recommendation`, `analyticsMode bool`, `analyticsScrollOffset int` fields to `App`
- [x] In `Update()` `SnapshotMsg` handler, store `app.recommendations = msg.Recommendations`
- [x] Add `Analytics key.Binding` to `keyMap` in `keys.go` (key: `"a"`, help: `"analytics"`)
- [x] Update help text to include `a: analytics`
- [x] In `Update()` `tea.KeyMsg` handler: toggle `analyticsMode` on `keys.Analytics`; reset scroll on mode switch; `esc` also exits analytics mode; ↑↓ scroll `analyticsScrollOffset` when in analytics mode
- [x] Run `make test` — must pass before task 5

### Task 5: Add analytics renderer

**Files:**
- Create: `internal/tui/analytics.go`
- Modify: `internal/tui/app.go`

- [x] Implement `renderAnalytics(app *App) string` in `analytics.go`
- [x] Title bar: `"Analytics — Cluster Recommendations"` with `[a/esc: back]` right-aligned
- [x] Group recommendations by category (ResourcePressure, ShardHealth, IndexConfig, Hotspot); render category header + items
- [x] Each item: severity badge (`[CRITICAL]`/`[WARN]`/`[OK]`) colored red/yellow/green + title + detail (detail wraps to next line, indented)
- [x] Show `"No issues found — cluster looks healthy"` in green when list is empty
- [x] Apply `analyticsScrollOffset` for scrolling; show scroll hint when content exceeds terminal height
- [x] In `app.go` `View()`: when `analyticsMode == true`, render header + analytics + footer
- [x] Run `make test` — must pass before task 6

### Task 6: Verify acceptance criteria

- [ ] Manual test: connect to ES cluster, press `a`, verify resource-aware recommendations appear with correct severity colors
- [ ] Verify a 4GB-heap cluster with 300+ shards shows a shard ratio critical/warning
- [ ] Verify `esc` and `a` both return to dashboard
- [ ] Verify ↑↓ scrolling when recommendations exceed screen height
- [ ] Verify "No issues found" message on a healthy cluster
- [ ] Run full test suite: `make test`
- [ ] Run linter: `make lint`

### Task 7: Update documentation

- [ ] Update `CLAUDE.md` project structure: add `analytics.go` to `tui/`, `recommendation.go` to `model/`, `recommendations.go` to `engine/`
- [ ] Update `CLAUDE.md` Keyboard Shortcuts table: add `a` → Toggle Analytics screen
- [ ] Move this plan to `docs/plans/completed/`
