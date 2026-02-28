# epm — Elasticsearch Performance Monitor

[![CI](https://github.com/jtsunne/epm-go/actions/workflows/ci.yml/badge.svg)](https://github.com/jtsunne/epm-go/actions/workflows/ci.yml)

Terminal dashboard for Elasticsearch cluster performance monitoring. A standalone Go binary that connects to any ES cluster and renders a live, interactive TUI — no browser required.

```
┌──────────────────────────────────────────────────────────────────┐
│ my-cluster  ● GREEN  Connected  Last: 14:32:05  Poll: 10s        │
├──────────────────────────────────────────────────────────────────┤
│ GREEN │ 5 Nodes │ 42 Idx │ 210 Shards │ CPU 34% │ JVM 67% │ S 45%│
├──────────────────────────────────────────────────────────────────┤
│ Indexing Rate  │ Search Rate  │ Index Latency  │ Search Latency  │
│  1,204.3 /s    │   892.1 /s   │   2.34 ms      │   5.67 ms       │
│  ▁▂▃▅▇█▇▅▃▂    │  ▁▃▅▇▅▃▁▂▃   │  ▁▁▂▂▃▃▂▁▁▁    │  ▁▂▃▂▁▁▂▃▄▃     │
├──────────────────────────────────────────────────────────────────┤
│ Index Statistics  [/: search]  [1-9: sort col]  [←→: page] 1/5   │
│ Name           │ P/T  │ Size  │ Shard   │  Docs  │Idx/s │Srch/s│ │
├──────────────────────────────────────────────────────────────────┤
│ Node Statistics                                         Page 1/1 │
│ Name       │ Role │ IP          │ Idx/s │ Srch/s │ ILat │  SLat  │
├──────────────────────────────────────────────────────────────────┤
│ tab: switch  /: search  1-9: sort  ←→: pages  r: refresh  q: quit│
└──────────────────────────────────────────────────────────────────┘
```

## Installation

**Go install (recommended):**

```bash
go install github.com/jtsunne/epm-go/cmd/epm@latest
```

**Build from source:**

```bash
git clone https://github.com/jtsunne/epm-go
cd epm-go
make build
# binary at bin/epm
```

**Homebrew (macOS and Linux):**

```bash
brew tap jtsunne/tap
brew install --cask epm
```

**Download binary:** pre-built binaries for macOS and Linux are available on the [Releases](https://github.com/jtsunne/epm-go/releases) page.

## Usage

```bash
epm [flags] <elasticsearch-uri>
```

The URI supports embedded credentials:

```bash
# Local cluster, no auth
epm http://localhost:9200

# Remote cluster with basic auth
epm https://elastic:changeme@es.prod.example.com:9200

# Passwords with special characters (#, ?, %) — use flags to bypass URL parsing
epm --user root --password "ab0102##" https://host:9200
epm --insecure --user root --password "s3cr#t!" https://prod.example.com:9200

# Credentials via environment variables (use https:// or add --allow-insecure-auth for http://)
ES_USER=elastic ES_PASSWORD=changeme epm --allow-insecure-auth http://localhost:9200
ES_PASSWORD="ab0102##" epm --user root https://host:9200

# Credential priority: --user/--password flags > ES_USER/ES_PASSWORD env vars > URI-embedded

# Custom poll interval (5s–300s)
epm --interval 30s http://localhost:9200

# Skip TLS verification for self-signed certificates
epm --insecure https://elastic:changeme@es.internal:9200

# Print version
epm --version
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--interval` | `10s` | Poll interval (5s–300s) |
| `--insecure` | false | Skip TLS certificate verification |
| `--user` | — | Elasticsearch username (overrides URI credentials and `ES_USER`) |
| `--password` | — | Elasticsearch password (overrides URI credentials and `ES_PASSWORD`) |
| `--allow-insecure-auth` | false | Allow sending credentials over unencrypted HTTP (not recommended for production) |
| `--version` | — | Print version and exit |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ES_USER` | Elasticsearch username. Overridden by `--user` flag. |
| `ES_PASSWORD` | Elasticsearch password. Overridden by `--password` flag. |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `r` | Force refresh immediately |
| `Tab` / `Shift+Tab` | Switch focused table |
| `↑` / `k` | Move cursor up in focused table |
| `↓` / `j` | Move cursor down in focused table |
| `1`–`9` | Sort by column N |
| `/` | Search in focused table |
| `Esc` | Close search |
| `←` / `→` | Previous / next page |
| `?` | Toggle help footer |
| `a` | Toggle Analytics screen (in analytics mode: `↑`/`↓` scroll, `a`/`Esc` return to dashboard) |
| `Space` | Toggle selection on focused index row (multi-select) |
| `d` | Delete selected index(es) — opens confirmation screen |

## Index Deletion

Press `Space` to toggle selection on rows in the index table (a `✓` prefix marks selected rows). Multiple rows can be selected. Press `d` to open the deletion confirmation screen.

The confirmation screen lists all indices pending deletion with a `WARNING: This action cannot be undone.` message. Press `y` to confirm or `n`/`Esc` to cancel.

After a successful deletion the index list refreshes automatically. The footer briefly shows `Deleted N index(es)` on success or `Delete failed: <reason>` on error.

`Space` and `d` only operate when the index table is focused. They have no effect on the node table or when search mode is active.

## Metrics Explained

**Indexing Rate** — new documents indexed per second, measured across primary shards only. Spikes here indicate bulk ingestion.

**Search Rate** — queries executed per second across all shards (primaries + replicas). Measures read traffic.

**Index Latency** — average time in milliseconds to complete one indexing operation during the last poll interval. High values (> 500 ms) indicate indexing pressure.

**Search Latency** — average time in milliseconds to complete one search query during the last poll interval. High values (> 1000 ms) degrade user-facing search experience.

**CPU %** — average `os.cpu.percent` across all data nodes (zero-percent nodes excluded from average).

**JVM Heap %** — average heap usage (`heap_used / heap_max`) across all nodes. Values above 75% indicate GC pressure; above 85% risks OOM.

**Storage %** — cluster-wide ratio of used disk space to total disk capacity across all nodes.

All rate and latency metrics are interval-based (delta between two consecutive polls), not cumulative totals. On the first poll cycle, rate and latency values display as `---` because a delta requires two consecutive snapshots; real values appear after the second poll.

## Alert Thresholds

Overview cards change color when thresholds are crossed — no alert history or management panel, purely visual.

| Metric | Warning (yellow) | Critical (red) |
|--------|-----------------|----------------|
| CPU | > 80% | > 90% |
| JVM Heap | > 75% | > 85% |
| Storage | > 80% | > 90% |
| Search Latency | — | > 1000 ms |
| Index Latency | > 500 ms | — |

Critical state adds a `!` suffix to the value and turns the card border red.

## Analytics Screen

Press `a` to switch from the dashboard to the Analytics screen. The screen shows a list of actionable recommendations derived from the current cluster snapshot.

Recommendations are grouped into five categories:

| Category | What it checks |
|----------|----------------|
| Resource Pressure | CPU, JVM heap, storage, and data-to-heap ratio |
| Shard Health | Cluster status (red/yellow), unassigned shards, shard-to-heap ratio, single data node |
| Index Configuration | Indices without replicas, oversized shards (> 50 GB), over-sharding (avg shard < 1 GB) |
| Hotspot | Uneven JVM heap utilization across nodes (spread > 30 pp) |
| Index Lifecycle | Date-patterned indices suitable for rollup consolidation (daily/weekly/monthly); empty deletion candidates |

Each recommendation is labelled `[CRITICAL]`, `[WARN]`, or `[OK]` (informational impact summary). When no issues are found, the screen shows "No issues found — cluster looks healthy".

Use `↑`/`↓` or `j`/`k` to scroll. Press `a` or `Esc` to return to the dashboard.

## Elasticsearch Version Compatibility

Tested with ES 6.x, 7.x, 8.x, and 9.x. All five API endpoints used are stable across these versions:

- `GET /_cluster/health` — cluster status and shard counts
- `GET /_cat/nodes?format=json` — node roles and IPs
- `GET /_nodes/stats/indices,os,jvm,fs` — per-node CPU, JVM, disk, and indexing stats
- `GET /_cat/indices?format=json` — per-index size and document counts
- `GET /_stats` — cluster-wide indexing and search operation totals

`filter_path` is used on all endpoints to minimize response payload size.

## Development

```bash
make build      # compile to bin/epm
make test       # go test -race -count=1 ./...
make lint       # go vet + staticcheck
make run ARGS="http://localhost:9200"
make clean      # remove bin/

# Integration tests (require a live ES cluster)
make integration ES_URI=http://localhost:9200
```

The project uses only the Go standard library plus [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI and `golang.org/x/sync/errgroup` for parallel fetching. No official Elasticsearch client dependency.

### Project Structure

```
cmd/epm/main.go        entry point, flag parsing
internal/client/       HTTP client and ES response types
internal/engine/       parallel fetch and metric calculation
internal/model/        snapshot, metrics, and sparkline history
internal/tui/          Bubble Tea model, renderers, and styles
internal/format/       number/byte/latency formatters
docs/plans/completed/  implementation plans (phases 1–7)
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-improvement`)
3. Add tests for any new logic (`make test` must pass with `-race`)
4. Open a pull request

## License

MIT. See [LICENSE](LICENSE) for details.
