# Phase 4: Metric Cards + Sparklines

## Overview

Add the 4 cluster-level metric cards (Indexing Rate, Search Rate, Index Latency, Search Latency) with inline braille-character sparklines showing historical trends. Uses the ring buffer history from Phase 2 and renders in the space between the overview bar and the tables.

## Context

- Chrome extension reference: `src/components/charts/MetricCard.tsx` and `src/components/charts/SparklineChart.tsx`
- Color coding from extension:
  - Indexing Rate: green (`#10b981`)
  - Search Rate: cyan (`#06b6d4`)
  - Index Latency: amber (`#f59e0b`)
  - Search Latency: red (`#ef4444`)
- Sparkline uses braille block chars `▁▂▃▄▅▆▇█` (same approach as many CLI tools like `btop`)
- `model.SparklineHistory` already implemented in Phase 2
- `model.PerformanceMetrics` and formatters already exist
- Depends on Phase 2 (model, formatters) and Phase 3 (TUI shell)

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Sparkline renderer is a pure function — easy to test without TUI
- Metric card layout: pure function of metric value + sparkline data

## Testing Strategy

- `TestRenderSparkline`: various value distributions, width constraints, all-zeros, single value
- `TestRenderMetricCards`: verify formatted values appear in output for known inputs
- No color/escape code assertions — test plain text content only

## Implementation Steps

### Task 1: Braille sparkline renderer

- [x] create `internal/tui/sparkline.go` with `RenderSparkline(values []float64, width int) string`:
  - use 8-level block chars: `var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}`
  - if `len(values) == 0` → return `strings.Repeat(" ", width)`
  - take last `width` values if slice longer than width
  - find `maxVal = slices.Max(values)` (use `math/slices` or manual loop)
  - if `maxVal == 0`, return all `▁` (floor level, not blank)
  - for each value: `idx = int(value / maxVal * 7)`, clamp to [0,7], write `sparkBlocks[idx]`
  - left-pad with spaces if fewer values than width
- [x] apply color to the sparkline string using the metric's lipgloss color style
- [x] write `internal/tui/sparkline_test.go`:
  - `TestRenderSparkline_Empty` — width=10 → 10 spaces
  - `TestRenderSparkline_AllZeros` — all chars are `▁`
  - `TestRenderSparkline_Ascending` — chars increase left to right
  - `TestRenderSparkline_TruncatesLeft` — 20 values, width=10 → uses last 10
  - `TestRenderSparkline_SingleValue` — 1 value → fills with spaces left, one `█` right
- [x] run tests — all pass

### Task 2: Single metric card renderer

- [x] create `internal/tui/metrics.go` with `renderMetricCard(title, value, unit string, sparkValues []float64, cardWidth int, color lipgloss.Color) string`:
  - card structure (3 rows):
    - row 1: title in dim/muted color (e.g. "Indexing Rate")
    - row 2: large value in bold + unit (e.g. "1,204.3 /s")
    - row 3: colored sparkline (full card width)
  - card has a subtle border (`lipgloss.RoundedBorder()`) and padding `0 1`
  - fixed card height to align all 4 cards
  - value colored with the metric's assigned color

### Task 3: 4-card metrics row renderer

- [x] implement `renderMetricsRow(app *App) string` in `internal/tui/metrics.go`:
  - returns empty string if `app.current == nil`
  - card width = `(app.width - 8) / 4` (account for borders/gaps)
  - card 1: **Indexing Rate** — `format.FormatRate(app.metrics.IndexingRate)`, sparkline from `app.history.Values("indexingRate")`, color `#10b981`
  - card 2: **Search Rate** — `format.FormatRate(app.metrics.SearchRate)`, color `#06b6d4`
  - card 3: **Index Latency** — `format.FormatLatency(app.metrics.IndexLatency)`, color `#f59e0b`
  - card 4: **Search Latency** — `format.FormatLatency(app.metrics.SearchLatency)`, color `#ef4444`
  - join 4 cards horizontally with `lipgloss.JoinHorizontal(lipgloss.Top, cards...)`
- [x] section label "Cluster Performance" above the card row in dim text

### Task 4: Wire sparkline history into App

- [x] update `App.Update()` in `app.go`: on `SnapshotMsg`, call `app.history.Push(model.SparklinePoint{...})` with the computed metrics
- [x] ensure `app.history` is initialized in `NewApp()` with cap 60
- [x] verify that after 3+ poll cycles, sparkline renders with non-space characters

### Task 5: Wire metrics row into View

- [x] update `App.View()` in `app.go` to call `renderMetricsRow(app)` and include it between overview and content area placeholder
- [x] handle narrow terminals: if `app.width < 80`, render metrics in 2x2 grid instead of 1x4 row

### Task 6: Tests for metric card logic

- [x] create `internal/tui/metrics_test.go`
- [x] `TestRenderMetricCard_ContainsValue` — output contains the formatted value string
- [x] `TestRenderMetricCard_ContainsTitle` — output contains the title string
- [x] `TestRenderMetricsRow_NilSnapshot` — returns empty string when app.current==nil
- [x] run `go test ./internal/tui/...` — all pass

### Task 7: Final verification

- [ ] run `go test ./...` — all pass
- [ ] run `go vet ./...` — no issues
- [ ] launch against live cluster: metric cards appear, values update each poll
- [ ] after 5+ polls: sparklines show growth, different shapes for different metrics
- [ ] test at terminal width 80 (2x2 layout) and 120+ (1x4 layout)

## Technical Details

- Sparkline height = 1 character row (embedded directly in the card, not a separate row)
- Braille blocks chosen over box-drawing characters for more granular levels (8 vs 4)
- `model.SparklineHistory.Values("indexingRate")` returns `[]float64` ready for `RenderSparkline`
- Metric cards use `lipgloss.RoundedBorder()` for modern look
- Card min width: 20 characters to fit title + value

## Post-Completion

- Sparklines should show visually distinct patterns when indexing rate is high vs low
- Verify no layout overflow at minimum 80-column terminal
