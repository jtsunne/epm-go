# Phase 5: Index + Node Tables

## Overview

Add the two sortable, paginated, searchable data tables: Index Statistics (9 columns) and Node Statistics (7 columns). Tables support keyboard navigation: column sorting by pressing a number key, pagination with arrow keys, and inline search filtering.

## Context

- Chrome extension reference: `src/components/data/IndexTable.tsx` (9 cols) and `NodeTable.tsx` (7 cols)
- Column definitions and color coding from those components
- `model.IndexRow` and `model.NodeRow` already produced by Phase 2 engine
- `lipgloss/table` for styled static table rendering with `StyleFunc` per-cell colors
- `charmbracelet/bubbles/textinput` for the search input field
- Depends on Phase 2 (IndexRow, NodeRow types), Phase 3 (App model), Phase 4 (view layout)

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Build generic table component first, then specialize for index and node tables
- Focus tests on sorting logic and filtering — not on rendered lipgloss output

## Testing Strategy

- `TestSortRows_ByIndexingRate` — sorting produces correct order
- `TestSortRows_Descending` — reverse order correct
- `TestFilterIndexRows_ByName` — search filter returns only matching rows
- `TestPaginate` — correct page slice for various inputs
- `TestRenderIndexTable_ContainsData` — verify index name appears in rendered output
- Avoid testing exact column widths or color escape codes

## Implementation Steps

### Task 1: Generic table model

- [ ] add `github.com/charmbracelet/lipgloss` table import (already added in Phase 3, but verify `lipgloss/table` subpackage)
- [ ] `go get github.com/charmbracelet/bubbles` (verify textinput available)
- [ ] create `internal/tui/table.go` with `tableModel` struct:
  ```go
  type tableModel struct {
      columns   []columnDef
      sortCol   int       // -1 = unsorted
      sortDesc  bool
      page      int       // 0-indexed
      pageSize  int       // default 10
      search    string
      searching bool
      input     textinput.Model
      focused   bool
  }

  type columnDef struct {
      Title    string
      Width    int
      Align    string  // "left", "right", "center"
      Key      string  // sort key
  }
  ```
- [ ] implement `tableModel.Update(msg tea.Msg) (tableModel, tea.Cmd)`:
  - digit keys `1`-`9` → set sort column; if same column, toggle direction
  - `←`/`h` → previous page
  - `→`/`l` → next page
  - `/` → enter search mode (start textinput)
  - `esc` → exit search, clear search term if empty
  - `enter` in search → apply filter, exit input
  - while `searching`: delegate key events to `textinput.Update()`
- [ ] implement `pageCount(totalRows, pageSize int) int` helper
- [ ] implement `currentPageRows(allRows []int, page, pageSize int) []int` — returns slice of indices

### Task 2: Sorting and filtering helpers

- [ ] create `internal/tui/sort.go` (or within table.go):
  - `sortIndexRows(rows []model.IndexRow, col int, desc bool) []model.IndexRow`
    - col 0 = Name (string asc default), col 1 = PrimaryShards, col 2 = TotalSizeBytes, col 3 = AvgShardSize, col 4 = DocCount, col 5 = IndexingRate, col 6 = SearchRate, col 7 = IndexLatency, col 8 = SearchLatency
    - default sort: col 5 (IndexingRate) desc
  - `sortNodeRows(rows []model.NodeRow, col int, desc bool) []model.NodeRow`
    - col 0 = Name, col 1 = Role, col 2 = IP, col 3 = IndexingRate, col 4 = SearchRate, col 5 = IndexLatency, col 6 = SearchLatency
    - default sort: col 3 (IndexingRate) desc
  - `filterIndexRows(rows []model.IndexRow, search string) []model.IndexRow`
    - case-insensitive substring match on `Name`
  - `filterNodeRows(rows []model.NodeRow, search string) []model.NodeRow`
    - match on `Name` or `IP`
- [ ] write `internal/tui/sort_test.go`:
  - `TestSortIndexRows_ByRate` — verify descending sort by IndexingRate
  - `TestSortIndexRows_ByName` — verify ascending alphabetical
  - `TestSortIndexRows_ToggleDirection` — same col twice → reverses
  - `TestFilterIndexRows_CaseInsensitive`
  - `TestFilterNodeRows_ByIP`
  - `TestSortNodeRows_BySearchRate`
- [ ] run tests — all pass

### Task 3: Index table renderer

- [ ] create `internal/tui/indextable.go`:
  - `IndexTableModel` embedding `tableModel` with typed row data `[]model.IndexRow`
  - `NewIndexTable() IndexTableModel` — initializes with 9 column defs:
    - 0: "Index Name" left, Width 25
    - 1: "P/T" center, Width 7 (primary/total shards)
    - 2: "Total Size" right, Width 10
    - 3: "Shard Size" right, Width 10
    - 4: "Doc Count" right, Width 12
    - 5: "Idx/s" right, Width 8 — green
    - 6: "Srch/s" right, Width 8 — blue
    - 7: "Idx Lat" right, Width 9 — purple
    - 8: "Srch Lat" right, Width 9 — orange
  - `SetData(rows []model.IndexRow)` — applies search filter, sort, stores result
  - `renderTable(app *App) string` — renders with `lipgloss/table`:
    - header row: bold, underlined, muted color; active sort column highlighted in blue
    - cell `StyleFunc`: col 5 → green, col 6 → blue, col 7 → purple, col 8 → orange (from `styles.go`)
    - alternating row backgrounds for readability
    - format values: `FormatBytes(TotalSizeBytes)`, `FormatRate(IndexingRate)`, `FormatLatency(IndexLatency)`, etc.
    - show "P/T" as "5/10" format
  - `renderHeader(title string, page, pageCount int, searching bool, searchTerm string) string`:
    - "Index Statistics  [/: search]  [1-9: sort]  [←→: page]   Page N/M"
    - if searching: show textinput inline
  - `renderFooter()` — brief column legend if help is shown

### Task 4: Node table renderer

- [ ] create `internal/tui/nodetable.go`:
  - `NodeTableModel` embedding `tableModel` with typed `[]model.NodeRow`
  - `NewNodeTable() NodeTableModel` — 7 column defs:
    - 0: "Node Name" left, Width 20
    - 1: "Role" left, Width 6 — blue badge style
    - 2: "IP" left, Width 15
    - 3: "Idx/s" right, Width 8 — green
    - 4: "Srch/s" right, Width 8 — blue
    - 5: "Idx Lat" right, Width 9 — purple
    - 6: "Srch Lat" right, Width 9 — orange
  - `SetData(rows []model.NodeRow)` — filter + sort
  - `renderTable(app *App) string` — same pattern as index table
  - Role column: render `dimr` style as colored abbreviated label (blue foreground, short)
  - Node role abbreviation tooltip shown in status bar when node row focused (Phase 6)

### Task 5: Wire tables into App model

- [ ] add `indexTable IndexTableModel` and `nodeTable NodeTableModel` to `App` struct
- [ ] add `activeTable int` (0 = index, 1 = node) and `focused bool`
- [ ] initialize both tables in `NewApp()`
- [ ] on `SnapshotMsg`: call `app.indexTable.SetData(msg.IndexRows)` and `app.nodeTable.SetData(msg.NodeRows)`
- [ ] in `App.Update()`:
  - `tab` / `shift+tab` → cycle `activeTable`
  - delegate key events to the active table's `Update()` method
- [ ] in `App.View()`: call `app.indexTable.renderTable()` and `app.nodeTable.renderTable()`, join vertically

### Task 6: Layout with height budget

- [ ] calculate available height for tables: `app.height - headerH - overviewH - metricsH - footerH`
- [ ] split remaining height between index table and node table (60%/40% or configurable)
- [ ] ensure tables truncate rows to fit available height rather than scrolling the whole screen
- [ ] if terminal height < 40, show only one table at a time (toggle with tab)

### Task 7: Tests for table rendering

- [ ] create `internal/tui/indextable_test.go`:
  - `TestIndexTableSetData_AppliesDefaultSort` — after SetData, rows ordered by IndexingRate desc
  - `TestIndexTableSearch` — after search="logs", only "logs-*" indices returned
  - `TestIndexTablePagination` — 25 rows, pageSize=10 → pageCount=3
  - `TestIndexTableRender_ContainsIndexName` — rendered string contains first index name
- [ ] create `internal/tui/nodetable_test.go`:
  - `TestNodeTableSetData_SortByRate`
  - `TestNodeTableSearch_ByIP`
- [ ] run `go test ./internal/tui/...` — all pass

### Task 8: Final verification

- [ ] run `go test ./...` — all pass
- [ ] run `go vet ./...` — no issues
- [ ] launch against live cluster:
  - index table shows correct data for all indices
  - press `5` → sort by indexing rate descending
  - press `1` → sort by name ascending
  - press `/` → search field appears, filter works
  - press `esc` → search closed
  - press `→` → next page, `←` → back
  - press `tab` → focus switches to node table
  - node table has correct node count

## Technical Details

- `lipgloss/table` `StyleFunc` receives `(row, col int)` and returns `lipgloss.Style` — use this for per-column color coding
- Row 0 in `StyleFunc` is the header row
- Sort default: numeric columns (rates, latencies) → descending; text columns (name) → ascending
- Secondary sort: ties in primary sort broken by index name (alphabetical)
- `pageSize` = 10 (matches Chrome extension default)
- Sort state persists between data refreshes (same column stays sorted)
- Search is applied BEFORE sort, sort BEFORE pagination

## Post-Completion

- Test with 100+ indices to verify pagination feels fast
- Test with 1 node (single-node cluster) — node table shows 1 row correctly
- Test search with special chars (`.`, `-`, `_` common in index names)
