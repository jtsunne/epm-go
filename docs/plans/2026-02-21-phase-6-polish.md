# Phase 6: Polish, Error Handling, and README

## Overview

Final phase: production-quality error states, threshold-based visual alerts on overview cards, graceful terminal resize, context cancellation on quit, and complete documentation. After this phase the binary is fully usable and ready for distribution.

## Context

- Chrome extension alert reference: `src/config/alerts.ts` — 13 rules with thresholds (we implement basic visual indicators only, not the full alert engine)
- Alert thresholds to implement as visual indicators:
  - CPU > 90% → card turns red; > 80% → yellow
  - JVM Heap > 85% → red; > 75% → yellow
  - Storage > 90% → red; > 80% → yellow
  - Search Latency > 1000ms → red metric card label
  - Indexing Latency > 500ms → yellow metric card label
- Error state reference: `src/components/feedback/ErrorState.tsx`
- Reconnection backoff already scaffolded in Phase 3 — this phase hardens it
- Depends on all previous phases

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- This phase is mostly integration-level testing + manual verification
- Tests focus on threshold logic (pure functions) and CLI flag parsing

## Testing Strategy

- `TestThreshold_CPU` — verify correct color selected at 0%, 79%, 80%, 89%, 90%, 100%
- `TestThreshold_JVM` — same boundary tests
- `TestThreshold_Storage` — same
- `TestThreshold_Latency` — both index and search latency thresholds
- `TestParseURI_UserPass` — verify user:pass extracted correctly from various URI formats
- `TestParseURI_NoCredentials` — URI without credentials → empty user/pass
- `TestParseURI_HTTPS_AutoInsecure` — when --insecure flag + https:// URI
- Manual: test full error→reconnect cycle

## Implementation Steps

### Task 1: Threshold helper functions

- [x] create `internal/tui/thresholds.go`:
  ```go
  type severity int
  const (
      severityNormal   severity = iota
      severityWarning           // yellow
      severityCritical          // red
  )

  func cpuSeverity(pct float64) severity       // >90 → critical, >80 → warning
  func jvmSeverity(pct float64) severity       // >85 → critical, >75 → warning
  func storageSeverity(pct float64) severity   // >90 → critical, >80 → warning
  func searchLatSeverity(ms float64) severity  // >1000 → critical
  func indexLatSeverity(ms float64) severity   // >500 → warning
  func severityToStyle(s severity) lipgloss.Style
  ```
- [x] write `internal/tui/thresholds_test.go`:
  - table-driven tests for each function with boundary values (79/80, 84/85, 89/90)
  - verify exactly the right severity at each threshold
- [x] run tests — all pass

### Task 2: Apply thresholds to overview cards

- [x] update `renderOverview()` in `overview.go`:
  - CPU card: apply `cpuSeverity(app.resources.AvgCPUPercent)` → color the percentage value
  - JVM card: apply `jvmSeverity(app.resources.AvgJVMHeapPercent)` → color the value
  - Storage card: apply `storageSeverity(app.resources.StoragePercent)` → color the percentage
  - Warning → yellow foreground on the value; Critical → red foreground + add "!" suffix
  - Card border also changes color on critical severity: use `lipgloss.RoundedBorder()` with red color

### Task 3: Apply thresholds to metric cards

- [x] update `renderMetricsRow()` in `metrics.go`:
  - Index Latency card: apply `indexLatSeverity(app.metrics.IndexLatency)` → title turns yellow on warning
  - Search Latency card: apply `searchLatSeverity(app.metrics.SearchLatency)` → title turns red on critical
  - Normal state: title in dim/muted color (as before)

### Task 4: Improved error display

- [x] update `renderHeader()` in `header.go` for disconnected state:
  - show error type distinctly: "Connection refused", "Authentication failed (401)", "Timeout", "TLS error"
  - extract error classification in `classifyError(err error) string`
  - if err contains "certificate" or "tls" → add SSL hint: "Try --insecure for self-signed certs"
  - show countdown to next retry: "Retrying in Ns... (r to retry now)"
  - implement countdown via `time.Until(app.nextRetryAt)` stored in App struct
- [x] add `nextRetryAt time.Time` to `App` struct, set it when scheduling backoff retry
- [x] update `App.Update()` to send a 1-second `tea.Tick` when disconnected (to update countdown display)

### Task 5: Terminal resize handling

- [x] verify `tea.WindowSizeMsg` handler in `app.go` re-renders all components
- [x] update `renderOverview()`: if `app.width < 80`, stack cards vertically (2 per row) instead of 1×7
- [x] update `renderMetricsRow()`: if `app.width < 80`, show 2×2 grid instead of 1×4
- [x] update tables: recalculate column widths on resize (clamp to min widths, drop columns if too narrow)
- [x] add `columnWidths(available int, defs []columnDef) []int` function that distributes width proportionally
- [x] test manually: resize terminal while running, verify no layout artifacts

### Task 6: Context cancellation and clean shutdown

- [x] update `fetchCmd` to create a context with `app.pollInterval - 500ms` timeout
- [x] when `tea.Quit` is sent (q key): Bubble Tea automatically cancels any pending commands
- [x] verify no goroutine leak: all `FetchAll` goroutines respect context cancellation
- [x] add graceful shutdown: if fetch is in-flight when quit pressed, it cancels within 10s max
- [x] verify terminal is fully restored after quit (no leftover alternate screen artifacts)

### Task 7: `--insecure` auto-suggestion

- [x] update `cmd/epm/main.go`:
  - if URI scheme is `https://` and `--insecure` not set: note in startup info (not an error, just a hint)
  - if connection fails with TLS error AND scheme is https: print "Hint: add --insecure flag for self-signed certificates" to stderr
  - `--insecure` automatically enables `InsecureSkipVerify` in the TLS config

### Task 8: CLI flag refinements

- [x] add `--version` flag: prints `epm version 0.1.0` and exits
- [x] improve usage message: show full example with all flags
- [x] validate `--interval`: must be between 5s and 300s; exit with error if out of range
- [x] if URI missing: show usage + "Error: ES URI is required" and exit 1
- [x] test URI parsing edge cases:
  - `http://localhost:9200` — no credentials
  - `https://elastic:changeme@es.prod.example.com:9200` — with credentials
  - `http://user:p%40ss@host:9200` — URL-encoded password (% escaping)
- [x] write `TestParseURI` table-driven test covering these cases

### Task 9: Makefile finalization

- [ ] update `Makefile`:
  ```makefile
  .PHONY: build test lint clean run

  build:
  	go build -ldflags="-X main.version=$(shell git describe --tags --always --dirty)" -o bin/epm ./cmd/epm

  test:
  	go test -race -count=1 ./...

  lint:
  	go vet ./...
  	@which staticcheck && staticcheck ./... || echo "staticcheck not installed, skipping"

  clean:
  	rm -rf bin/

  run:
  	go run ./cmd/epm $(ARGS)

  integration:
  	ES_URI=$(ES_URI) go test -tags=integration ./...
  ```
- [ ] add version variable in `cmd/epm/main.go`: `var version = "dev"` (overridden by ldflags)
- [ ] `--version` flag prints the version

### Task 10: README.md

- [ ] create `README.md` at project root:
  - Project title + brief description ("Terminal dashboard for Elasticsearch cluster performance monitoring")
  - Screenshot/demo section (ASCII art placeholder or actual terminal screenshot)
  - **Installation**: `go install github.com/dm/epm-go/cmd/epm@latest`, or Homebrew (future), or download binary
  - **Usage**: `epm <uri> [flags]` with examples for all scenarios
  - **Keyboard shortcuts**: table of all keybindings (from `keys.go`)
  - **Metrics explained**: what each metric means (indexing rate, search rate, latency)
  - **ES version compatibility**: tested with ES 6.x, 7.x, 8.x, 9.x
  - **Alert thresholds**: table of when overview cards change color
  - **Development**: `make build`, `make test`, contributing guide
  - **License**: MIT

### Task 11: Final integration test file

- [ ] create `internal/integration_test.go` (build tag: `//go:build integration`):
  - `TestLiveCluster_AllEndpoints` — connects to `$ES_URI`, calls FetchAll, verifies non-empty snapshot
  - `TestLiveCluster_MetricsNonNegative` — runs 2 polls, verifies rates >= 0
  - `TestLiveCluster_HTTPSWithInsecure` — if URI is https://, verify TLS skip-verify works
  - Run with: `make integration ES_URI=http://localhost:9200`

### Task 12: Final verification pass

- [ ] run `go test -race ./...` — all pass, no race conditions
- [ ] run `go vet ./...` — clean
- [ ] run `go build -o bin/epm ./cmd/epm` — clean build
- [ ] manual testing scenarios:
  - [ ] fresh ES 8.x cluster: all metrics display correctly
  - [ ] cluster with 0 indexing activity: rates show 0, no panic
  - [ ] cluster with many indices (50+): pagination works
  - [ ] kill ES while running: disconnected state shown, retries, recovers
  - [ ] `--insecure` with self-signed cert ES: connects successfully
  - [ ] resize terminal to 80 cols: layout adapts, no overflow
  - [ ] resize terminal to 200 cols: layout expands, no wasted space
  - [ ] quit with q: terminal fully restored, cursor visible
- [ ] verify against ES 6.x compatibility (key: `_cat` JSON format uses same field names)

## Technical Details

- Race condition check: `go test -race` is important because of goroutines in fetchCmd
- Version injection: `-ldflags="-X main.version=..."` via git describe
- Integration tests are skipped by default (no `//go:build integration` tag = excluded from `go test ./...`)
- Self-signed TLS: `tls.Config{InsecureSkipVerify: true}` only when `--insecure` explicitly set

## Post-Completion

- Consider publishing binary releases via GitHub Actions
- Consider Homebrew formula for macOS users
- ES 6.x note: `_cat/nodes` JSON format has same field names, but some fields may be absent — verify graceful nil handling
