# Phase 2: Polling Engine + Metric Calculations

## Overview

Build the business logic layer: parallel 5-endpoint fetching, snapshot model, delta-based rate/latency calculations, cluster resource aggregation, and number formatters. This is the most mathematically critical phase — all metric calculations must be exact ports of the Chrome extension logic.

## Context

- Source of truth for calculations: `/Users/dm/dev/elasticsearch-performance-monitoring/src/utils/performanceTracker.ts`
- Per-index metric source: `/Users/dm/dev/elasticsearch-performance-monitoring/src/components/data/IndexTable.tsx` (lines 66-131)
- Per-node metric source: `/Users/dm/dev/elasticsearch-performance-monitoring/src/components/data/NodeTable.tsx` (lines 76-113)
- Cluster resource aggregation: `/Users/dm/dev/elasticsearch-performance-monitoring/src/App.tsx` (lines 193-240)
- Formatters: `/Users/dm/dev/elasticsearch-performance-monitoring/src/utils/format.ts`
- Depends on Phase 1 (`internal/client/`)

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Calculator tests are the highest-value tests in the project — invest in comprehensive coverage
- Use table-driven tests with named cases

## Testing Strategy

- `calculator_test.go`: table-driven tests with fixture snapshots for all metric formulas
- Test nil-prev case (first snapshot, no previous → rates = 0)
- Test counter reset (current < previous → clamp to 0, not negative)
- Test sanity caps (rate > 50M/s → 0, latency > 300s → cap)
- Test minimum time diff (< 1s → return zeros)
- Test primaries-vs-total rule for per-index indexing/search
- `poller_test.go`: mock ESClient tests for FetchAll success + partial failure
- `format_test.go`: table-driven tests for all formatters

## Implementation Steps

### Task 1: Snapshot and model types

- [x] create `internal/model/snapshot.go`:
  ```go
  type Snapshot struct {
      Health     client.ClusterHealth
      Nodes      []client.NodeInfo
      NodeStats  client.NodeStatsResponse
      Indices    []client.IndexInfo
      IndexStats client.IndexStatsResponse
      FetchedAt  time.Time
  }
  ```
- [x] create `internal/model/metrics.go`:
  ```go
  type PerformanceMetrics struct {
      IndexingRate  float64  // ops/sec
      SearchRate    float64  // ops/sec
      IndexLatency  float64  // ms/op
      SearchLatency float64  // ms/op
  }
  type ClusterResources struct {
      AvgCPUPercent     float64
      AvgJVMHeapPercent float64
      StorageUsedBytes  int64
      StorageTotalBytes int64
      StoragePercent    float64
  }
  type NodeRow struct {
      ID, Name, Role, IP string
      IndexingRate, SearchRate float64
      IndexLatency, SearchLatency float64
  }
  type IndexRow struct {
      Name                    string
      PrimaryShards, TotalShards int
      TotalSizeBytes, AvgShardSize int64
      DocCount                int64
      IndexingRate, SearchRate float64
      IndexLatency, SearchLatency float64
  }
  ```
- [x] create `internal/model/history.go`:
  - `SparklinePoint` struct: `Timestamp time.Time`, `IndexingRate`, `SearchRate`, `IndexLatency`, `SearchLatency float64`
  - `SparklineHistory` with fixed-size ring buffer (default cap 60)
  - methods: `Push(SparklinePoint)`, `Values(field string) []float64`, `Len() int`, `Clear()`
  - `Values` accepts field names `"indexingRate"`, `"searchRate"`, `"indexLatency"`, `"searchLatency"`
- [x] write tests for `SparklineHistory`: push beyond capacity overwrites oldest, `Values` returns correct slice

### Task 2: FetchAll parallel poller

- [x] add `golang.org/x/sync` dependency: `go get golang.org/x/sync`
- [x] create `internal/engine/poller.go`:
  ```go
  func FetchAll(ctx context.Context, c client.ESClient) (*model.Snapshot, error)
  ```
  - use `errgroup.WithContext(ctx)` to launch 5 goroutines concurrently
  - each goroutine calls one `ESClient` method
  - if any goroutine fails, errgroup cancels the context and `Wait()` returns the first error
  - on success, return assembled `*model.Snapshot` with `FetchedAt: time.Now()`
- [x] create `internal/engine/mock_client_test.go` — `MockESClient` struct implementing `ESClient` interface with configurable return values
- [x] write `TestFetchAll_AllSuccess` — all 5 methods return valid data, verify snapshot fields
- [x] write `TestFetchAll_PartialFailure` — one method returns error, verify FetchAll returns error
- [x] write `TestFetchAll_ContextCancelled` — pre-cancelled context, verify error
- [x] run `go test ./internal/engine/...` — all pass

### Task 3: Sanity constants and helpers

- [x] create `internal/engine/calculator.go` with constants (ported from `performanceTracker.ts` lines 96-100):
  ```go
  const (
      minTimeDiffSeconds = 1.0
      maxRatePerSec      = 50_000_000.0
      maxLatencyMs       = 300_000.0
  )
  ```
- [x] implement helper functions:
  - `clampRate(r float64) float64` — returns 0 if r > maxRatePerSec, else r
  - `clampLatency(l float64) float64` — caps at maxLatencyMs
  - `safeDivide(a, b float64) float64` — returns 0 if b==0
  - `maxFloat64(a, b float64) float64`

### Task 4: Cluster-level metrics calculation

- [x] implement `CalcClusterMetrics(prev, curr *model.Snapshot, elapsed time.Duration) model.PerformanceMetrics`:
  - if prev==nil or elapsed < minTimeDiffSeconds → return zero PerformanceMetrics
  - aggregate totals across ALL nodes in curr and prev NodeStats.Nodes
  - `indexOpsDelta = max(0, currTotalIndexOps - prevTotalIndexOps)`
  - `searchOpsDelta = max(0, currTotalSearchOps - prevTotalSearchOps)`
  - apply `clampRate` and `clampLatency`
  - latency formula: `deltaTimeMs / deltaOps` — not cumulative
- [x] write table-driven tests in `internal/engine/calculator_test.go`:
  - `TestCalcClusterMetrics_FirstSnapshot` — prev=nil → all zeros
  - `TestCalcClusterMetrics_BasicRates` — known delta, verify rate = delta/elapsed
  - `TestCalcClusterMetrics_CounterReset` — current ops < previous → rate = 0
  - `TestCalcClusterMetrics_TooShortInterval` — elapsed < 1s → all zeros
  - `TestCalcClusterMetrics_RateSanityCap` — huge delta → rate = 0
  - `TestCalcClusterMetrics_LatencySanityCap` — huge latency → capped

### Task 5: Cluster resource aggregation

- [x] implement `CalcClusterResources(snap *model.Snapshot) model.ClusterResources`:
  - CPU: average `os.cpu.percent` across nodes, filtering zeros (same as App.tsx lines 193-209)
  - JVM: per-node `heap_used / heap_max * 100`, average across nodes, filtering zeros (App.tsx lines 211-222)
  - Storage: sum `total_in_bytes - available_in_bytes` across all nodes; `StoragePercent = used/total*100` (App.tsx lines 224-238)
  - handle nil OS/JVM/FS fields gracefully
- [x] write `TestCalcClusterResources`:
  - verify CPU average skips nodes with 0%
  - verify JVM heap percentage calculation
  - verify storage sum and percentage

### Task 6: Per-index metrics calculation

- [x] implement `CalcIndexRows(prev, curr *model.Snapshot, elapsed time.Duration) []model.IndexRow`:
  - iterate `curr.Indices` to build rows
  - for each index, look up `curr.IndexStats.Indices[name]` and `prev.IndexStats.Indices[name]`
  - **CRITICAL rule** (from IndexTable.tsx lines 73-76 and `types/api.ts` comment):
    - indexing ops/time: use `primaries` (fallback to `total` if primaries nil)
    - search ops/time: use `total` (fallback to `primaries` if total nil)
  - `TotalSizeBytes` from `total.store.size_in_bytes`
  - `AvgShardSize = PrimarySizeBytes / max(1, PrimaryShards)`
  - `TotalShards = pri * (1 + rep)` (parse string fields `Pri`, `Rep` to int)
  - `DocCount`: parse `DocsCount` string to int64
  - apply same rate/latency/clamp formulas as cluster-level
- [x] write `TestCalcIndexRows`:
  - `TestCalcIndexRows_PrimariesForIndexing` — indexing uses primaries, not total
  - `TestCalcIndexRows_TotalForSearch` — search uses total, not primaries
  - `TestCalcIndexRows_NilPrev` — returns rows with zero rates
  - `TestCalcIndexRows_ShardCountParsing` — "5" pri, "1" rep → TotalShards=10

### Task 7: Per-node metrics calculation

- [x] implement `CalcNodeRows(prev, curr *model.Snapshot, elapsed time.Duration) []model.NodeRow`:
  - build map `nodeID → NodePerformanceStats` for curr and prev
  - build map `nodeName → NodeInfo` from `curr.Nodes` (for role/IP lookup)
  - for each node in `curr.NodeStats.Nodes`: compute delta rates using same formula
  - if node not in prev (new node), return zero rates for that node
- [x] write `TestCalcNodeRows`:
  - verify rates computed per node correctly
  - verify node not in prev → zero rates (no crash)
  - verify role and IP populated from Nodes list

### Task 8: Number formatters

- [x] create `internal/format/format.go`:
  - `FormatBytes(bytes int64) string` — B/KB/MB/GB/TB with 1 decimal place (port from `format.ts` formatBytes)
  - `FormatRate(opsPerSec float64) string` — e.g. "1,204.3 /s", "0 /s"
  - `FormatLatency(ms float64) string` — e.g. "2.34 ms", "1.50 s" (>=1000ms → seconds)
  - `FormatNumber(n int64) string` — locale-style comma separator e.g. "12,345,678"
  - `FormatPercent(p float64) string` — e.g. "34.5%"
  - `ParseHumanBytes(s string) int64` — parse "20.4gb" → bytes (for `pri.store.size` from _cat/indices)
- [x] create `internal/format/format_test.go` with table-driven tests for all formatters:
  - `FormatBytes`: 0, 1023, 1024, 1536, 1GB, edge cases
  - `FormatLatency`: <1000ms shows ms, >=1000ms shows seconds
  - `FormatRate`: 0, 1.0, 1204.3, large numbers with comma formatting
  - `ParseHumanBytes`: "50gb", "20.4gb", "100mb", "1.5tb"
- [x] run `go test ./internal/format/...` — all pass

### Task 9: Wire engine into CLI for debug output

- [x] update `cmd/epm/main.go` to run two sequential FetchAll calls (10s apart) and print computed metrics to stdout
- [x] verify rates are non-zero when cluster has activity
- [x] verify `go build ./cmd/epm` — compiles

### Task 10: Final verification

- [x] run `go test ./...` — all tests pass
- [x] run `go vet ./...` — no issues
- [x] given two fixture snapshots with known deltas, manually verify rates match hand calculations

## Technical Details

- All counters are `int64` to handle large cluster cumulative values without overflow
- `elapsed` is computed as `curr.FetchedAt.Sub(prev.FetchedAt)` (actual time, not poll interval)
- Per-index and per-node calculations are done in the engine layer, not in the TUI layer
- `ParseHumanBytes` needed because `_cat/indices` returns human-readable sizes like "20.4gb"

## Post-Completion

- Verify calculations against Chrome extension output on same cluster
- Spot-check: if extension shows indexing rate = X, binary should show same X (within 5% timing variance)
