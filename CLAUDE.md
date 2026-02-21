# epm-go — Elasticsearch Performance Monitor TUI

Go TUI reimplementation of the Chrome extension "Elasticsearch Performance Monitoring". Standalone binary that connects to an ES cluster and renders a live terminal dashboard.

## Usage

```bash
epm [--interval 10s] [--insecure] <elasticsearch-uri>

# Examples
epm http://localhost:9200
epm --insecure https://elastic:changeme@prod.example.com:9200
epm --interval 30s http://localhost:9200
```

## Tech Stack

| Layer | Library |
|-------|---------|
| TUI framework | `charmbracelet/bubbletea` (MVU, `tea.Tick` for polling) |
| Styling | `charmbracelet/lipgloss` |
| Tables | `charmbracelet/lipgloss/table` (StyleFunc per-cell coloring) |
| Text input | `charmbracelet/bubbles/textinput` |
| Sparklines | Braille chars `▁▂▃▄▅▆▇█` — zero-dependency |
| HTTP client | `net/http` stdlib — no official ES client |
| Parallel fetch | `golang.org/x/sync/errgroup` |
| CLI flags | `stdlib flag` — no Cobra |
| Testing | `stretchr/testify` + `net/http/httptest` |

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
  engine/
    poller.go                # FetchAll: 5 endpoints in parallel via errgroup
    calculator.go            # CalcClusterMetrics, CalcClusterResources, CalcNodeRows, CalcIndexRows
    calculator_test.go       # Table-driven tests for all metric formulas
    poller_test.go
  tui/
    app.go                   # Root bubbletea Model: Init/Update/View
    messages.go              # SnapshotMsg, FetchErrorMsg, TickMsg
    keys.go                  # Key bindings (q, r, tab, /, 1-9, ←→)
    styles.go                # All lipgloss styles and color constants
    header.go                # Header bar renderer
    footer.go                # Footer bar renderer (help text / key hints)
    overview.go              # 7-stat overview bar renderer
    metrics.go               # 4 metric cards row renderer          (Phase 4)
    sparkline.go             # RenderSparkline(values, width) string (Phase 4)
    table.go                 # Generic tableModel: sort, paginate, search (Phase 5)
    indextable.go            # IndexTableModel (9 columns)           (Phase 5)
    nodetable.go             # NodeTableModel (7 columns)            (Phase 5)
    thresholds.go            # Threshold severity functions for alert coloring (Phase 6)
  format/
    format.go                # FormatBytes, FormatRate, FormatLatency, FormatNumber, FormatPercent
    format_test.go
docs/plans/                  # Ralphex implementation plans (phases 1-6)
Makefile
```

## Elasticsearch API Endpoints

All 5 endpoints are GET-only, JSON, with `filter_path` to reduce response size. Stable across ES 6.x–9.x.

```
GET /_cluster/health?filter_path=cluster_name,status,number_of_nodes,active_shards
GET /_cat/nodes?v&format=json&h=node.role,name,ip&s=node.role,ip
GET /_nodes/stats/indices,os,jvm,fs?filter_path=nodes.*.name,...
GET /_cat/indices?v&format=json&h=index,pri,rep,pri.store.size,store.size,docs.count&s=index
GET /_stats?filter_path=indices.*.primaries.indexing...,indices.*.total.search...
```

Full `filter_path` values are in `internal/client/endpoints.go`.

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
│ Name       │ Role │ IP          │ Idx/s │ Srch/s │ ILat │ SLat  │  ← nodetable.go
├──────────────────────────────────────────────────────────────────┤
│ tab: switch  /: search  1-9: sort  ←→: pages  r: refresh  q: quit│  ← keys.go
└──────────────────────────────────────────────────────────────────┘
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q` / `ctrl+c` | Quit |
| `r` | Force refresh now |
| `tab` / `shift+tab` | Switch focused table |
| `1`–`9` | Sort by column N |
| `/` | Search in focused table |
| `esc` | Close search |
| `←` / `→` | Previous / next page |
| `?` | Toggle help footer |

## Color Coding

| Metric | Color |
|--------|-------|
| Indexing Rate | Green `#10b981` |
| Search Rate | Cyan `#06b6d4` |
| Index Latency | Amber `#f59e0b` |
| Search Latency | Red `#ef4444` |
| Cluster GREEN | Emerald |
| Cluster YELLOW | Amber |
| Cluster RED | Red (entire bg tints red) |

## Development

```bash
make build    # go build -o bin/epm ./cmd/epm
make test     # go test -race -count=1 ./...
make lint     # go vet ./... + staticcheck
make run ARGS="http://localhost:9200"

# Integration tests (require live ES cluster)
make integration ES_URI=http://localhost:9200
```

## Testing Conventions

- `internal/client/client_test.go` — httptest server returning fixture JSON for each of the 5 endpoints
- `internal/engine/calculator_test.go` — table-driven tests with fixture Snapshots; highest-value tests in project
- `internal/format/format_test.go` — table-driven tests for all formatters
- `internal/model/history_test.go` — ring-buffer overflow and Values() correctness for SparklineHistory
- `internal/tui/*_test.go` — test Update() logic and pure render functions; do NOT assert on lipgloss color escape codes
- Integration tests: `//go:build integration`, skipped by default
- Mock ESClient: `internal/engine/mock_client_test.go` (implements `client.ESClient` interface)

## Architecture Notes

- **Bubble Tea MVU**: all state in `App` struct; mutations only in `Update()`; no goroutine-level shared state
- **Snapshot rotation**: each `SnapshotMsg` moves `current → previous`, sets new `current`
- **fetchCmd context**: timeout = `max(pollInterval - 500ms, 500ms)`; cancelled automatically on quit
- **Backoff**: `min(2^consecutiveFails * second, 60s)` starting at 2s
- **`_cat` string fields**: `IndexInfo.Pri`, `Rep`, `DocsCount` are strings from the API — parse in `CalcIndexRows`, not in the client
- **Never store credentials** beyond the lifetime of the process (in-memory `ClientConfig` only)

## ES Version Compatibility

All 5 endpoints are stable across ES 6.x, 7.x, 8.x, 9.x:
- `_cat` JSON format (`?format=json`) available since ES 5.0
- `filter_path` available since ES 1.6
- No version detection or version-specific code paths


