# Add Index Deletion Feature

## Overview

Add the ability to delete one or multiple Elasticsearch indices from the TUI. The user navigates to an index row and presses `d` to delete it (single-row mode), or uses `space` to toggle selection on multiple rows and then presses `d` to delete all selected. A confirmation screen always appears before any deletion is performed. After deletion, the index list is refreshed automatically.

## Context

- Files involved:
  - `internal/client/client.go` — ESClient interface
  - `internal/client/endpoints.go` — HTTP methods
  - `internal/client/client_test.go` — client tests
  - `internal/engine/mock_client_test.go` — mock client used in engine tests
  - `internal/tui/keys.go` — key bindings
  - `internal/tui/indextable.go` — IndexTableModel (multi-select state + rendering)
  - `internal/tui/app.go` — App state and Update handler
  - `internal/tui/messages.go` — message types
  - `internal/tui/footer.go` — shows delete status after operation
  - `internal/tui/delete.go` (new) — renderDeleteConfirm + deleteCmd
- Related patterns: analytics mode (new UI mode replacing main content), fetchCmd pattern for async commands, SnapshotMsg / FetchErrorMsg message typing
- Dependencies: none new

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Add DeleteIndex to ESClient and DefaultClient

**Files:**
- Modify: `internal/client/client.go`
- Modify: `internal/client/endpoints.go`
- Modify: `internal/client/client_test.go`
- Modify: `internal/engine/mock_client_test.go`

- [x] Add `DeleteIndex(ctx context.Context, names []string) error` to `ESClient` interface in `client.go`
- [x] Add `doDelete(ctx, path string) error` helper to `DefaultClient` in `endpoints.go` (no response body needed — ES returns JSON but we only care about success/failure)
- [x] Implement `DeleteIndex` on `DefaultClient`: builds path `/<comma-joined names>`, calls `doDelete`; wrap error with `fmt.Errorf("DeleteIndex: %w", err)`
- [x] Add `DeleteIndexFn func(ctx context.Context, names []string) error` field to `MockESClient` in `mock_client_test.go`; implement `DeleteIndex` method that calls it when non-nil, otherwise returns nil
- [x] Add httptest-based tests in `client_test.go`: handler returns 200 OK (success), handler returns 404 (error surfaced), batch path contains comma-separated names
- [x] run `make test` and `make lint` — must pass

### Task 2: Multi-select support in IndexTableModel

**Files:**
- Modify: `internal/tui/keys.go`
- Modify: `internal/tui/indextable.go`
- Create: `internal/tui/indextable_test.go`

- [x] Add `ToggleSelect` key binding (space) and `DeleteKey` key binding (d) to `keyMap` struct and `keys` var in `keys.go`
- [x] Add `selected map[string]struct{}` field to `IndexTableModel`; initialize to `make(map[string]struct{})` in `NewIndexTable()`
- [x] Add `toggleSelect(name string)` method: adds if absent, removes if present
- [x] Add `selectedNames() []string` method: returns sorted slice of selected index names
- [x] In `IndexTableModel.Update`: intercept `keys.ToggleSelect` (space) — toggle selection for the row under cursor on current page, then return (do not pass to tableModel)
- [x] In `renderTable`: mark selected rows with `"✓ "` prefix on the name cell (col 0); in the `StyleFunc` close over the selected set and apply `colorIndigo` background to selected rows (cursor row still gets `colorSelectedBg` on top)
- [x] Clear `selected` in `SetData` when the underlying data changes (prevents stale selections after a refresh removes an index)
- [x] Write table-driven tests in `indextable_test.go`: toggleSelect adds/removes names, selectedNames returns sorted list, SetData clears selections, space key in Update toggles row under cursor
- [x] run `make test` and `make lint` — must pass

### Task 3: Delete confirmation flow in App

**Files:**
- Modify: `internal/tui/messages.go`
- Create: `internal/tui/delete.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/footer.go`
- Modify: `internal/tui/keys.go` (helpText constant)
- Create: `internal/tui/delete_test.go`

- [x] Add `DeleteResultMsg{Names []string, Err error}` to `messages.go`
- [x] Create `delete.go`:
  - `deleteCmd(c client.ESClient, names []string) tea.Cmd`: issues DELETE via `c.DeleteIndex` in a goroutine, returns `DeleteResultMsg`
  - `renderDeleteConfirm(app *App) string`: styled full-screen confirmation view listing pending indices with "Press y to confirm, n/esc to cancel"; reuse `StyleHeader` for the title bar; list index names in body area; styled with `StyleRed` warning text
- [x] Add state fields to `App` struct in `app.go`: `deleteConfirmMode bool`, `pendingDeleteNames []string`, `deleteStatus string`
- [x] In `App.Update` — handle `keys.DeleteKey`: only when `activeTable == 0` and NOT in searching mode; collect `indexTable.selectedNames()`; if empty and cursor row exists use cursor row name; if target list non-empty set `pendingDeleteNames` and `deleteConfirmMode = true`
- [x] In `App.Update` — when `deleteConfirmMode` is true, handle only: `y` → execute `deleteCmd`, exit confirm mode, clear selection; `n`/`esc` → exit confirm mode, clear `pendingDeleteNames`; block all other keys
- [x] In `App.Update` — handle `DeleteResultMsg`: on success set `deleteStatus = "Deleted N index(es)"`, trigger immediate `fetchCmd` refresh; on error set `deleteStatus = fmt.Sprintf("Delete failed: %v", msg.Err)`
- [x] Clear `deleteStatus` when a new `SnapshotMsg` arrives (fetch completed)
- [x] In `App.View`: when `deleteConfirmMode` is true, render header + `renderDeleteConfirm(app)` + footer (similar to analytics mode)
- [x] In `footer.go`: when `app.deleteStatus != ""`, show it in the footer (red for error, green for success); detect by checking if status starts with "Delete failed"
- [x] Update `helpText` constant in `keys.go` to include `space: select  d: delete`
- [x] Write `delete_test.go`: test `renderDeleteConfirm` renders index names; test that `deleteCmd` returns correct `DeleteResultMsg` on success and error (using httptest or mock); test App.Update transitions in/out of deleteConfirmMode on y/n/esc
- [x] run `make test` and `make lint` — must pass

### Task 4: Verify acceptance criteria

- [x] manual test: start epm against a local ES cluster; navigate index table; press space on 2 rows; press d; confirm y; verify indices are gone after refresh
- [x] manual test: press d without selection; confirm y; verify single index deleted
- [x] manual test: press n in confirmation; verify no deletion occurred
- [x] run `make test` — all tests pass
- [x] run `make lint` — no new warnings
- [x] run `make build` — binary compiles cleanly

### Task 5: Update documentation

- [x] Update `CLAUDE.md` keyboard shortcuts table: add space (toggle select) and d (delete index)
- [x] Move this plan to `docs/plans/completed/`
