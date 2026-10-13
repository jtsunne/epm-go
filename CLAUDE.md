# epm-go — Elasticsearch Performance Monitor TUI

Go TUI reimplementation of the Chrome extension "Elasticsearch Performance Monitoring". Standalone binary that connects to an ES cluster and renders a live terminal dashboard.

## Usage

```bash
epm [--interval 10s] [--insecure] [--user USER] [--password PASS] [--theme auto|light|dark] <elasticsearch-uri>

# Examples
epm http://localhost:9200
epm --insecure https://elastic:changeme@prod.example.com:9200
epm --interval 30s http://localhost:9200

# Color theme: auto-detects terminal background by default; force when detection is wrong (tmux/screen/SSH)
epm --theme light http://localhost:9200

# Credentials with special characters (# ? % etc.) — use flags instead of URI
epm --user root --password "op0107##" https://host:9200
epm --insecure --user root --password "s3cr#t!" https://prod.example.com:9200

# Credentials via environment variables (http:// requires --allow-insecure-auth)
ES_USER=elastic ES_PASSWORD=changeme epm --allow-insecure-auth http://localhost:9200
ES_PASSWORD="op0107##" epm --user root https://host:9200

# Credential priority: --user/--password flags > ES_USER/ES_PASSWORD env vars > URI-embedded
```

## Project Structure

```
cmd/epm/main.go              # Entry point: flag parsing, URI parsing, tea.NewProgram
internal/
  client/
    client.go                # ESClient interface + DefaultClient (TLS, BasicAuth, timeout)
    endpoints.go             # 5 endpoint path constants with filter_path
    types.go                 # Raw ES JSON response structs
    client_test.go           # httptest-based tests
  model/
    snapshot.go              # Snapshot: result of one poll cycle
    metrics.go               # PerformanceMetrics, ClusterResources, NodeRow, IndexRow
    history.go               # SparklineHistory ring buffer (cap 60)
    recommendation.go        # Recommendation type, severity/category constants
  engine/
    poller.go                # FetchAll: 6 endpoints in parallel via errgroup
    calculator.go            # CalcClusterMetrics, CalcClusterResources, CalcNodeRows, CalcIndexRows
    calculator_test.go       # Table-driven tests for all metric formulas
    poller_test.go
    recommendations.go       # CalcRecommendations: resource-aware cluster recommendations
    recommendations_test.go  # Table-driven tests for recommendation rules
  tui/
    app.go                   # Root bubbletea Model: Init/Update/View
    messages.go              # SnapshotMsg, FetchErrorMsg, TickMsg
    keys.go                  # Key bindings (q, r, tab, /, 1-9, ←→, ↑↓/j/k, a)
    styles.go                # All lipgloss styles and color constants
    header.go                # Header bar renderer
    footer.go                # Footer bar renderer (help text / key hints)
    overview.go              # 7-stat overview bar renderer
    metrics.go               # 4 metric cards row renderer          (Phase 4)
    sparkline.go             # RenderSparkline(values, width) string (Phase 4)
    table.go                 # Generic tableModel: sort, paginate, search, cursor navigation, name truncation (Phase 5, 7)
    sort.go                  # sortIndexRows, sortNodeRows, filterIndexRows, filterNodeRows (Phase 5)
    indextable.go            # IndexTableModel (9 columns)           (Phase 5)
    nodetable.go             # NodeTableModel (9 columns)            (Phase 5)
    thresholds.go            # Threshold severity functions for alert coloring (Phase 6)
    analytics.go             # renderAnalytics: full-screen recommendations view
    delete.go                # renderDeleteConfirm + deleteCmd (index deletion confirmation overlay)
    settings.go              # SettingsFormModel, settingsLoadCmd, settingsUpdateCmd, renderSettingsForm
  format/
    format.go                # FormatBytes, FormatRate, FormatLatency, FormatNumber, FormatPercent
    format_test.go
docs/plans/completed/        # Completed implementation plans (phases 1-7)
Makefile
```

## Elasticsearch API Endpoints

6 endpoints, all GET-only except the write path. Full paths and `filter_path` values are in `internal/client/endpoints.go`.

The PUT `/_settings` endpoint is the only write-mutating operation. `doPutJSON` on `DefaultClient` is the HTTP transport helper for PUT requests with a JSON body — it mirrors `doDelete` shape. Used exclusively by `UpdateIndexSettings`.

## Key Metric Formulas

```
Rate    = (current_total_ops - previous_total_ops) / elapsed_seconds
Latency = delta_time_ms / delta_ops   ← interval-based, NOT cumulative

Indexing metrics: use PRIMARIES only
Search metrics:   use TOTAL (all shards)

Cluster CPU%:     average os.cpu.percent across nodes (skip zeros)
Cluster JVM%:     average (heap_used / heap_max * 100) across nodes (skip zeros)
Cluster Storage:  sum (total - available) / sum total across all nodes
```

**Sanity caps**:
- Rate > 50,000,000 ops/s → set to 0
- Latency > 300,000 ms → cap at 300,000
- Minimum time between snapshots: 1 second

## Alert Thresholds (visual indicators only)

Overview cards change color when thresholds are exceeded — no alert history or management panel.

| Metric | Warning (yellow) | Critical (red) |
|--------|-----------------|----------------|
| CPU | > 80% | > 90% |
| JVM Heap | > 75% | > 85% |
| Storage | > 80% | > 90% |
| Search Latency | — | > 1000 ms |
| Index Latency | > 500 ms | — |

## TUI Layout

```
┌──────────────────────────────────────────────────────────────────┐
│ my-cluster  ● GREEN  Connected  Last: 14:32:05  Poll: 10s        │  ← header.go
├──────────────────────────────────────────────────────────────────┤
│ GREEN │ 5 Nodes │ 42 Idx │ 210 Shards │ CPU 34% │ JVM 67% │ S 45%│  ← overview.go
├──────────────────────────────────────────────────────────────────┤
│ Indexing Rate  │ Search Rate  │ Index Latency  │ Search Latency  │  ← metrics.go
│  1,204.3 /s   │   892.1 /s  │   2.34 ms      │   5.67 ms       │
│  ▁▂▃▅▇█▇▅▃▂   │  ▁▃▅▇▅▃▁▂▃ │  ▁▁▂▂▃▃▂▁▁▁   │  ▁▂▃▂▁▁▂▃▄▃    │  ← sparkline.go
├──────────────────────────────────────────────────────────────────┤
│ Index Statistics  [/: search]  [1-9: sort col]  [←→: page] 1/5  │
│ Name           │ P/T  │ Size   │ Shard │  Docs  │Idx/s│Srch/s│..│  ← indextable.go
├──────────────────────────────────────────────────────────────────┤
│ Node Statistics                                         Page 1/1  │
│ Name  │ Role │ IP       │ Idx/s │ Srch/s │ ILat │ SLat │Shards│Disk%│  ← nodetable.go
├──────────────────────────────────────────────────────────────────┤
│ tab: switch  /: search  1-9: sort  ←→: pages  r: refresh  q: quit│  ← keys.go
└──────────────────────────────────────────────────────────────────┘
```

Keyboard shortcuts are in `internal/tui/keys.go`; color constants for metric cards and table columns are in `internal/tui/styles.go`.

## Development

See `Makefile` for build/test/lint/run targets (including the `integration` target, which requires a live ES cluster).

## Pre-Commit Checklist

Before marking any task done or committing:
1. `make lint` — must pass with no new warnings
2. `make test` — all tests must pass
3. `make build` — binary must compile cleanly

Error handling: always wrap with `fmt.Errorf("context: %w", err)`.
Follow existing patterns in the file being modified before introducing new ones.

## Testing Conventions

- `cmd/epm/main_test.go` — table-driven tests for `parseESURI`: credential stripping, scheme validation, invalid host detection
- `internal/client/client_test.go` — httptest server returning fixture JSON for each of the 6 endpoints
- `internal/engine/calculator_test.go` — table-driven tests with fixture Snapshots; highest-value tests in project
- `internal/format/format_test.go` — table-driven tests for all formatters
- `internal/model/history_test.go` — ring-buffer overflow and Values() correctness for SparklineHistory
- `internal/tui/*_test.go` — test Update() logic and pure render functions; do NOT assert on lipgloss color escape codes
- Integration tests: `//go:build integration`, skipped by default
- Mock ESClient: `internal/engine/mock_client_test.go` (implements `client.ESClient` interface)

## Architecture Notes

- **Bubble Tea MVU**: all state in `App` struct; mutations only in `Update()`; no goroutine-level shared state
- **Snapshot rotation**: each `SnapshotMsg` moves `current → previous`, sets new `current`
- **fetchCmd context**: timeout = `clamp(pollInterval - 500ms, 500ms, 10s)`; the 10s cap ensures fast process exit on quit regardless of poll interval
- **Backoff**: `min(2^consecutiveFails * second, 60s)` starting at 2s
- **`_cat` string fields**: `IndexInfo.Pri`, `Rep`, `DocsCount` are strings from the API — parse in `CalcIndexRows`, not in the client
- **Never store credentials** beyond the lifetime of the process (in-memory `ClientConfig` only)
- **MetricNotAvailable sentinel**: `model.MetricNotAvailable = -1.0` is returned by all calculator functions when `prev == nil` or elapsed < 1s (delta not computable). `FormatRate` and `FormatLatency` treat any value `< 0` as sentinel and display `"---"`. Safe because the engine clamps all real metrics to `>= 0`. Sparkline history guard in `app.go` prevents sentinel values from entering the history buffer.
- **`CategoryIndexLifecycle`**: recommendation category (in `model/recommendation.go`) used for date-rollup suggestions (daily→weekly/monthly, weekly→monthly, monthly→yearly) and empty-index deletion candidates. Separate from `CategoryIndexConfig` to keep lifecycle vs. config concerns distinct.
- **`IndexRow.PriSizeBytes`**: primary shard data size in bytes (int64), populated by `CalcIndexRows` from the already-computed `primarySizeBytes` local variable. Used by `dateRollupRecs` to decide daily→monthly vs. daily→weekly consolidation target (threshold: 100 MiB average primary size per index).
- **`deleteConfirmMode`**: UI overlay mode (like `analyticsMode`) — when true, `View()` renders header + `renderDeleteConfirm` + footer instead of the dashboard. Only `y`, `n`, and `esc` are processed; all other keys are blocked. State fields: `deleteConfirmMode bool`, `pendingDeleteNames []string`, `deleteStatus string`, `deleteStatusErr bool`. `deleteStatus` is cleared on the next `SnapshotMsg` or `FetchErrorMsg`. `deleteStatusErr` controls footer color (red vs green).
- **`settingsMode`**: UI overlay mode (like `deleteConfirmMode`) — when true, `View()` renders header + `renderSettingsForm` + footer instead of the dashboard. Key events are routed to `SettingsFormModel.Update()`; `ctrl+s` submits changed fields via `settingsUpdateCmd` and exits settingsMode immediately (prevents double-submit); `esc` cancels. Field navigation: `↑`/`↓` and `Tab`/`Shift+Tab` are equivalent (Tab does NOT move focus outside the form). State fields: `settingsMode bool`, `settingsForm SettingsFormModel`, `settingsStatus string`, `settingsStatusErr bool`, `settingsPendingRefresh bool`. `settingsPendingRefresh` mirrors `pendingRefresh` for delete: protects `settingsStatus` from the stale in-flight SnapshotMsg when a fetch was already in-flight at the time the settings update completed. Node names/IPs for routing allocation suggestions are extracted from the current snapshot at form open time (no extra API call).

## lipgloss Layout Patterns

- **`.Width(n)` sets minimum width only** — pads with spaces but does NOT cap overflow. Use `.MaxWidth(n)` to truncate content that exceeds a boundary (e.g., safety net on header bar).
- **Equal width distribution**: `base = width / n; rem = width % n` — first `rem` elements get `base+1`. Ensures `sum(widths) == width` exactly with no trailing gap.
- **Card height equalization**: `.Height(n).AlignVertical(lipgloss.Center)` makes lipgloss fill the card background for all `n` lines and vertically centers shorter content — prevents mismatched background fill in `JoinHorizontal`.
- **Header overflow prevention**: progressive truncation order — truncate left (plain text) first, then right via `MaxWidth`, then hard-truncate center as last resort. Add `const minGap = 1` between segments. Always apply `.MaxWidth(innerWidth)` to the final rendered row as a safety net.

## ES Version Compatibility

All 6 endpoints are stable across ES 6.x, 7.x, 8.x, 9.x:
- `_cat/allocation` may return empty on some ES versions — handled gracefully (non-fatal, Shards/Disk% show "---")
- `_cat` JSON format (`?format=json`) available since ES 5.0
- `filter_path` available since ES 1.6
- No version detection or version-specific code paths


