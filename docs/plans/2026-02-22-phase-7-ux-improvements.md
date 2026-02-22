# Phase 7: UX Improvements — Initial Display & Table Navigation

## Overview

Two UX issues found during testing of the completed Phase 1-6 implementation:

1. **Initial display discrepancy**: After the first snapshot, metric cards and table rate/latency columns show "0 /s" / "0.00 ms" — semantically wrong. These aren't "zero operations", they're "not yet computed" (delta requires two snapshots). Users see misleading zeros for one full polling interval.

2. **Long index names overflow layout**: Index names like `logstash-production-2024.01.15-000042` exceed the computed column width, causing `lipgloss/table` to wrap cell text and increase row height. With many indices, the header/overview/metrics sections are pushed off screen.

**Solution**:
- Issue 1: sentinel value `MetricNotAvailable = -1.0` propagated from calculator → formatters → display as "---"
- Issue 2: name truncation with "..." + up/down cursor navigation + detail line showing full name

## Context

- All rate/latency metrics require a previous snapshot (delta). `CalcClusterMetrics`, `CalcNodeRows`, `CalcIndexRows` all return zero-value structs when `prev == nil`.
- `FormatRate` and `FormatLatency` treat negative values as sentinel → "---". Safe because engine clamps all real metrics to `>= 0`.
- Column 0 (Index Name) has preferred width 25 but is proportionally scaled; lipgloss/table wraps text when cell content exceeds column width.
- `computeTablePageSizes()` uses `tableOverhead = 3` (title + header + separator). Adding a detail line requires bumping to 4.
- Current `tableModel` has no cursor concept. Up/down keys are unbound.

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Testing Strategy

- `TestFormatRate_Sentinel` — `FormatRate(-1)` → `"---"`
- `TestFormatLatency_Sentinel` — `FormatLatency(-1)` → `"---"`
- `TestTruncateName` — table-driven: short, exact, long, unicode, width 0-3, empty
- `TestTableModel_CursorUpDown` — cursor moves within page bounds, clamped at 0
- `TestTableModel_CursorResetOnPageChange` — cursor resets to 0 on page nav
- `TestTableModel_CursorResetOnSearch` — cursor resets when search changes
- Update existing calculator tests to expect `MetricNotAvailable` where `prev == nil`

## Implementation Steps

### Task 1: Add MetricNotAvailable sentinel constant

**Files:**
- Modify: `internal/model/metrics.go`

- [x] add sentinel constant:
  ```go
  // MetricNotAvailable signals that a rate/latency metric has not yet been
  // computed (requires two snapshots for delta calculation).
  const MetricNotAvailable float64 = -1.0
  ```
- [x] run `go build ./...` — clean build

### Task 2: Return sentinel from calculator when prev is nil

**Files:**
- Modify: `internal/engine/calculator.go`
- Modify: `internal/engine/calculator_test.go`

- [x] update `CalcClusterMetrics()`: early-return branch returns all four fields as `model.MetricNotAvailable` instead of zero-value struct:
  ```go
  if prev == nil || curr == nil || elapsed.Seconds() < minTimeDiffSeconds {
      return model.PerformanceMetrics{
          IndexingRate:  model.MetricNotAvailable,
          SearchRate:    model.MetricNotAvailable,
          IndexLatency:  model.MetricNotAvailable,
          SearchLatency: model.MetricNotAvailable,
      }
  }
  ```
- [x] update `CalcNodeRows()`: when `!enoughTime`, assign sentinel to rate/latency fields of each row:
  ```go
  row.IndexingRate = model.MetricNotAvailable
  row.SearchRate = model.MetricNotAvailable
  row.IndexLatency = model.MetricNotAvailable
  row.SearchLatency = model.MetricNotAvailable
  ```
- [x] update `CalcIndexRows()`: same sentinel assignment when `!enoughTime`
- [x] update existing `calculator_test.go` table cases that expected `0` for nil-prev scenarios to expect `model.MetricNotAvailable`
- [x] run `go test ./internal/engine/...` — all pass

### Task 3: Handle sentinel in formatters

**Files:**
- Modify: `internal/format/format.go`
- Modify: `internal/format/format_test.go`

- [ ] update `FormatRate()`: add sentinel guard before existing logic:
  ```go
  func FormatRate(opsPerSec float64) string {
      if opsPerSec < 0 {
          return "---"
      }
      // ... existing code unchanged
  }
  ```
- [ ] update `FormatLatency()`: same guard:
  ```go
  func FormatLatency(ms float64) string {
      if ms < 0 {
          return "---"
      }
      // ... existing code unchanged
  }
  ```
  Rationale: engine already clamps real values to `>= 0`; negative is unreachable without the sentinel.
- [ ] add tests: `FormatRate(-1)` → `"---"`, `FormatLatency(-1)` → `"---"`
- [ ] verify existing tests unchanged: `FormatRate(0)` → `"0 /s"`, `FormatLatency(0)` → `"0.00 ms"`
- [ ] run `go test ./internal/format/...` — all pass

### Task 4: Add truncateName helper to table.go

**Files:**
- Modify: `internal/tui/table.go`
- Modify: `internal/tui/table_test.go`

- [ ] add `truncateName` function:
  ```go
  // truncateName truncates s to fit within maxWidth runes, appending "..."
  // if truncated. Returns s unchanged if it fits. Uses []rune for correct
  // Unicode handling. ES index names are ASCII in practice, but node names
  // can be arbitrary hostnames.
  func truncateName(s string, maxWidth int) string {
      runes := []rune(s)
      if len(runes) <= maxWidth {
          return s
      }
      if maxWidth <= 3 {
          if maxWidth <= 0 {
              return ""
          }
          return string(runes[:maxWidth])
      }
      return string(runes[:maxWidth-3]) + "..."
  }
  ```
- [ ] write table-driven tests covering: short (fits), exact width, one over, long name, very narrow (width 0, 1, 2, 3, 4), unicode, empty string; verify `len([]rune(got)) <= maxWidth` for all cases
- [ ] run `go test ./internal/tui/...` — all pass

### Task 5: Apply name truncation in index and node tables

**Files:**
- Modify: `internal/tui/indextable.go`
- Modify: `internal/tui/nodetable.go`

- [ ] update `indextable.go` `renderTable()`: after building `cells`, truncate column 0 to `colWidths[0]`:
  ```go
  for _, idx := range pageIdx {
      r := m.displayRows[idx]
      cells := make([]string, len(m.columns))
      for col := range m.columns {
          cells[col] = indexCellValue(r, col)
      }
      // Prevent cell wrapping: truncate name to allocated column width.
      if len(colWidths) > 0 && colWidths[0] > 0 {
          cells[0] = truncateName(cells[0], colWidths[0])
      }
      t = t.Row(cells...)
  }
  ```
- [ ] apply the same pattern in `nodetable.go` `renderTable()` for Node Name (column 0)
- [ ] run `go test ./internal/tui/...` — all pass

### Task 6: Add cursor state and up/down navigation

**Files:**
- Modify: `internal/tui/table.go`
- Modify: `internal/tui/keys.go`
- Modify: `internal/tui/indextable.go`
- Modify: `internal/tui/nodetable.go`

- [ ] add `cursor int` field to `tableModel` struct; init to `0` in `newTableModel()`
- [ ] add `clampCursor(pageRowCount int)` method to `tableModel`:
  ```go
  func (t *tableModel) clampCursor(pageRowCount int) {
      if pageRowCount <= 0 {
          t.cursor = 0
          return
      }
      if t.cursor >= pageRowCount {
          t.cursor = pageRowCount - 1
      }
      if t.cursor < 0 {
          t.cursor = 0
      }
  }
  ```
- [ ] add key bindings to `keys.go` keyMap struct and `keys` var:
  ```go
  CursorUp:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "cursor up")),
  CursorDown: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "cursor down")),
  ```
  Note: `j`/`k` only reach this handler when `!t.searching` — textinput already captures all keys in search mode.
- [ ] handle up/down in `tableModel.Update()` non-searching branch:
  ```go
  case key.Matches(msg, keys.CursorUp):
      if t.cursor > 0 {
          t.cursor--
      }
      return t, nil
  case key.Matches(msg, keys.CursorDown):
      t.cursor++
      return t, nil
  ```
- [ ] reset `cursor = 0` on: `PrevPage`, `NextPage`, `Enter` (search confirm), `Escape` (clear search), digit sort key
- [ ] call `clampCursor(pageRowCount)` in `IndexTableModel.Update()` after `clampPage()`, computing `pageRowCount` from displayRows slice bounds
- [ ] call `clampCursor(pageRowCount)` in `NodeTableModel.Update()` — same pattern
- [ ] call `clampCursor(pageRowCount)` in `IndexTableModel.SetData()` and `NodeTableModel.SetData()` after `clampPage()`
- [ ] update `helpText` constant in `keys.go`:
  ```go
  const helpText = "tab: switch table  /: search  1-9: sort col  ←→: pages  ↑↓: select row  r: refresh  q: quit  ?: close help"
  ```
- [ ] write tests: cursor up/down movement, clamp at 0 (can't go below), reset on page change, reset on search apply/clear, reset on sort change
- [ ] run `go test ./internal/tui/...` — all pass

### Task 7: Cursor highlight and detail line in tables

**Files:**
- Modify: `internal/tui/styles.go`
- Modify: `internal/tui/indextable.go`
- Modify: `internal/tui/nodetable.go`

- [ ] add selected row color to `styles.go`:
  ```go
  var colorSelectedBg = lipgloss.Color("#1e3a5f") // subtle blue highlight
  ```
- [ ] update `indextable.go` `renderTable()` `StyleFunc`: when `focused && row == cursor`, use `colorSelectedBg` background; otherwise keep existing alternating row logic:
  ```go
  focused := m.focused
  cursor := m.cursor
  StyleFunc(func(row, col int) lipgloss.Style {
      if row == ltable.HeaderRow { /* ... unchanged */ }
      base := lipgloss.NewStyle()
      if focused && row == cursor {
          base = base.Background(colorSelectedBg)
      } else if row%2 == 0 {
          base = base.Background(colorAlt)
      }
      // ... column colors unchanged
  })
  ```
- [ ] add detail line at bottom of `indextable.go` `renderTable()`: if focused and data present, show full untruncated name:
  ```go
  var detailLine string
  if m.focused && len(pageIdx) > 0 && m.cursor < len(pageIdx) {
      detailLine = StyleDim.Render("  " + m.displayRows[pageIdx[m.cursor]].Name)
  }
  if detailLine != "" {
      return lipgloss.JoinVertical(lipgloss.Left, hdr, t.String(), detailLine)
  }
  return lipgloss.JoinVertical(lipgloss.Left, hdr, t.String())
  ```
- [ ] apply same cursor highlight and detail line in `nodetable.go` `renderTable()`: detail line shows `Name + "  " + Role + "  " + IP`
- [ ] write tests: detail line contains full untruncated name when focused; detail line absent when unfocused
- [ ] run `go test ./internal/tui/...` — all pass

### Task 8: Adjust height budget for detail line

**Files:**
- Modify: `internal/tui/app.go`

- [ ] update `computeTablePageSizes()`: change comment and constant from 3 to 4 overhead lines:
  ```go
  // Each rendered table section costs:
  //   1 title bar line  +  1 column-header row  +  1 separator line  +  1 detail line = 4 overhead lines.
  const tableOverhead = 4
  ```
- [ ] run `go test ./internal/tui/...` — all pass

### Task 9: Final verification

- [ ] run `go test -race ./...` — all pass, no race conditions
- [ ] run `go vet ./...` — clean
- [ ] run `go build -o bin/epm ./cmd/epm` — clean build
- [ ] manual: first snapshot shows "---" for all rate/latency; second poll shows real values
- [ ] manual: cluster with 0 activity — after second poll shows "0 /s" (not "---")
- [ ] manual: long index names truncated with "...", row height stays 1 line
- [ ] manual: ↑↓ / j/k move cursor highlight within focused table
- [ ] manual: detail line shows full untruncated name below focused table
- [ ] manual: tab switches focus — cursor + detail line move to other table
- [ ] manual: page change / search / sort resets cursor to first row
- [ ] manual: terminal resize — cursor clamped, layout stable

### Task 10: Move plan to completed

- [ ] move `docs/plans/2026-02-22-phase-7-ux-improvements.md` → `docs/plans/completed/`

## Technical Details

- Sentinel `-1.0`: safe because all real rates/latencies are clamped to `>= 0` by the engine. `FormatRate`/`FormatLatency` now treat `< 0` as "not available". No changes needed at call sites in `indextable.go`, `nodetable.go`, or `metrics.go` — the formatter handles it.
- `truncateName` uses ASCII "..." (3 chars) not Unicode "…" (1 char) for predictable monospace terminal width.
- Cursor is **page-relative** (0-indexed within current page, not absolute across all rows). This matches the `row` parameter in `StyleFunc` directly.
- Detail line costs 1 vertical line per table → `tableOverhead` bumped from 3 to 4. Both tables budget for it; the unfocused table's "unused" line gives 1 extra data row (acceptable).
- `j`/`k` vim keys work because they only reach the cursor handler when `!t.searching`. In search mode, `textinput.Update()` captures all key events.

## Post-Completion

- Consider mouse click support for row selection (Bubble Tea supports `tea.MouseMsg`)
- Consider expanding the detail line to show additional stats on selection

## Critical Files

| File | Changes |
|------|---------|
| `internal/model/metrics.go` | Add `MetricNotAvailable` constant |
| `internal/engine/calculator.go` | Return sentinel when prev is nil |
| `internal/format/format.go` | Handle negative sentinel → "---" |
| `internal/tui/table.go` | Add cursor, clampCursor, truncateName, up/down handling |
| `internal/tui/keys.go` | Add CursorUp/CursorDown bindings, update helpText |
| `internal/tui/indextable.go` | Truncation, cursor highlight, detail line |
| `internal/tui/nodetable.go` | Truncation, cursor highlight, detail line |
| `internal/tui/styles.go` | Add `colorSelectedBg` |
| `internal/tui/app.go` | tableOverhead = 4 |
