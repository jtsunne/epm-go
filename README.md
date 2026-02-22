# epm — Elasticsearch Performance Monitor

Terminal dashboard for Elasticsearch cluster performance monitoring. A standalone Go binary that connects to any ES cluster and renders a live, interactive TUI — no browser required.

```
┌──────────────────────────────────────────────────────────────────┐
│ my-cluster  ● GREEN  Connected  Last: 14:32:05  Poll: 10s        │
├──────────────────────────────────────────────────────────────────┤
│ GREEN │ 5 Nodes │ 42 Idx │ 210 Shards │ CPU 34% │ JVM 67% │ S 45%│
├──────────────────────────────────────────────────────────────────┤
│ Indexing Rate  │ Search Rate  │ Index Latency  │ Search Latency  │
│  1,204.3 /s   │   892.1 /s  │   2.34 ms      │   5.67 ms       │
│  ▁▂▃▅▇█▇▅▃▂   │  ▁▃▅▇▅▃▁▂▃ │  ▁▁▂▂▃▃▂▁▁▁   │  ▁▂▃▂▁▁▂▃▄▃    │
├──────────────────────────────────────────────────────────────────┤
│ Index Statistics  [/: search]  [1-9: sort col]  [←→: page] 1/5  │
│ Name           │ P/T  │ Size   │ Shard │  Docs  │Idx/s│Srch/s│  │
├──────────────────────────────────────────────────────────────────┤
│ Node Statistics                                         Page 1/1  │
│ Name       │ Role │ IP          │ Idx/s │ Srch/s │ ILat │ SLat  │
├──────────────────────────────────────────────────────────────────┤
│ tab: switch  /: search  1-9: sort  ←→: pages  r: refresh  q: quit│
└──────────────────────────────────────────────────────────────────┘
```

## Installation

**Go install (recommended):**

```bash
go install github.com/dm/epm-go/cmd/epm@latest
```

**Build from source:**

```bash
git clone https://github.com/dm/epm-go
cd epm-go
make build
# binary at bin/epm
```

**Download binary:** pre-built binaries for macOS and Linux will be available on the [Releases](https://github.com/dm/epm-go/releases) page.

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
| `--version` | — | Print version and exit |

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `r` | Force refresh immediately |
| `Tab` / `Shift+Tab` | Switch focused table |
| `1`–`9` | Sort by column N |
| `/` | Search in focused table |
| `Esc` | Close search |
| `←` / `→` | Previous / next page |
| `?` | Toggle help footer |

## Metrics Explained

**Indexing Rate** — new documents indexed per second, measured across primary shards only. Spikes here indicate bulk ingestion.

**Search Rate** — queries executed per second across all shards (primaries + replicas). Measures read traffic.

**Index Latency** — average time in milliseconds to complete one indexing operation during the last poll interval. High values (> 500 ms) indicate indexing pressure.

**Search Latency** — average time in milliseconds to complete one search query during the last poll interval. High values (> 1000 ms) degrade user-facing search experience.

**CPU %** — average `os.cpu.percent` across all data nodes (zero-percent nodes excluded from average).

**JVM Heap %** — average heap usage (`heap_used / heap_max`) across all nodes. Values above 75% indicate GC pressure; above 85% risks OOM.

**Storage %** — cluster-wide ratio of used disk space to total disk capacity across all nodes.

All rate and latency metrics are interval-based (delta between two consecutive polls), not cumulative totals.

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
docs/plans/            implementation plans (phases 1–6)
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-improvement`)
3. Add tests for any new logic (`make test` must pass with `-race`)
4. Open a pull request

## License

MIT. See [LICENSE](LICENSE) for details.
