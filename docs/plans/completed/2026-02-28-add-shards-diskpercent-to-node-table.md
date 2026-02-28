# Add Shards and Disk% Columns to Node Table

## Overview

Add two new columns to the nodes table sourced from a new `/_cat/allocation` endpoint:
- `Shards` — number of shards allocated on the node (integer)
- `Disk%` — node disk usage percentage (float, formatted as "XX.X%")

The allocation data is fetched in parallel with the existing 5 endpoints and joined to node rows by node name in the calculator.

## Context

- Files involved:
  - `internal/client/types.go` — new AllocationInfo type
  - `internal/client/endpoints.go` — new endpoint constant + GetAllocation method
  - `internal/client/client.go` — extend ESClient interface
  - `internal/client/client_test.go` — new test
  - `internal/model/snapshot.go` — add Allocation field
  - `internal/model/metrics.go` — add Shards/DiskPercent to NodeRow
  - `internal/engine/poller.go` — add parallel GetAllocation fetch
  - `internal/engine/calculator.go` — populate Shards/DiskPercent in CalcNodeRows
  - `internal/engine/mock_client_test.go` — add AllocationFn + GetAllocation stub
  - `internal/engine/calculator_test.go` — new test cases
  - `internal/tui/nodetable.go` — add 2 columns, update cell renderer, update sort hint
  - `internal/tui/sort.go` — add sort cases 7 and 8
- Related patterns:
  - `_cat` string fields parsed in calculator (same as `IndexInfo.Pri`, `Rep`, etc.)
  - Sentinel value -1 for unavailable metrics displayed as "---"
  - Parallel fetch with errgroup in poller.go
  - Mock client for engine tests

## Development Approach

- **Testing approach**: Regular (code first, then tests within same task)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Client layer — Add `_cat/allocation` endpoint

**Files:**
- Modify: `internal/client/types.go`
- Modify: `internal/client/endpoints.go`
- Modify: `internal/client/client.go`
- Modify: `internal/client/client_test.go`

- [x] Add `AllocationInfo` struct to `types.go` with string fields: `Node`, `Shards`, `DiskPercent` (json: `"node"`, `"shards"`, `"disk.percent"`)
- [x] Add `endpointAllocation` constant to `endpoints.go`: `/_cat/allocation?format=json&h=node,shards,disk.percent&s=node`
- [x] Add `GetAllocation(ctx context.Context) ([]AllocationInfo, error)` method to `DefaultClient` in `endpoints.go`, following the same pattern as `GetNodes`
- [x] Add `GetAllocation` signature to `ESClient` interface in `client.go`
- [x] Add `TestGetAllocation` test in `client_test.go` using httptest, verifying path prefix `/_cat/allocation`, format=json, and correct field decoding
- [x] run `make test` — must pass before task 2

### Task 2: Model layer — Extend NodeRow and Snapshot

**Files:**
- Modify: `internal/model/metrics.go`
- Modify: `internal/model/snapshot.go`

- [x] Add `Shards int` field to `NodeRow` in `metrics.go` (sentinel: -1 = not in allocation data)
- [x] Add `DiskPercent float64` field to `NodeRow` in `metrics.go` (sentinel: -1.0 = not available)
- [x] Add `Allocation []client.AllocationInfo` field to `Snapshot` in `snapshot.go`
- [x] run `make build` to confirm no compilation errors

### Task 3: Engine layer — Fetch and populate allocation data

**Files:**
- Modify: `internal/engine/poller.go`
- Modify: `internal/engine/calculator.go`
- Modify: `internal/engine/mock_client_test.go`
- Modify: `internal/engine/calculator_test.go`

- [x] Add `GetAllocation` call to the `errgroup` in `FetchAll` (`poller.go`), storing result in local `allocation []client.AllocationInfo`; set `snap.Allocation = allocation`
- [x] Treat allocation as non-fatal — if nil/empty (some ES versions may not support it), proceed without error
- [x] In `CalcNodeRows` (`calculator.go`), build a `nodeNameToAlloc map[string]client.AllocationInfo` from `curr.Allocation` keyed by `AllocationInfo.Node`
- [x] For each `NodeRow`, look up allocation by `node.Name`; if found, parse `Shards` (strconv.Atoi → default -1 on error) and `DiskPercent` (strconv.ParseFloat → default -1.0 on error); if not found, set both to -1
- [x] Add `AllocationFn func(ctx context.Context) ([]client.AllocationInfo, error)` to `MockESClient` in `mock_client_test.go`, add `GetAllocation` method with default returning empty slice
- [x] Add test `TestCalcNodeRows_AllocationData` to `calculator_test.go`: snapshot with two nodes, one with valid allocation data, one missing; assert Shards and DiskPercent populated correctly and sentinel for missing node
- [x] run `make test` — must pass before task 4

### Task 4: TUI layer — Add columns to node table

**Files:**
- Modify: `internal/tui/nodetable.go`
- Modify: `internal/tui/sort.go`

- [x] In `NewNodeTable()`, add two columns after the existing 7: `{Title: "Shards", Width: 7, SortDesc: true}` and `{Title: "Disk%", Width: 7, SortDesc: true}`; update comment from "7-column" to "9-column"
- [x] In `nodeCellValue`, add case 7 (Shards: if < 0 show "---" else `strconv.Itoa(r.Shards)`) and case 8 (DiskPercent: if < 0 show "---" else `format.FormatPercent(r.DiskPercent)`)
- [x] In `StyleFunc` inside `renderTable`, add color cases: col 7 → `colorWhite`, col 8 → `colorYellow`
- [x] In `renderHeader`, update sort hint from `[1-7: sort]` to `[1-9: sort]`
- [x] In `sortNodeRows` (`sort.go`): add case 7 sorting by `Shards` (int, with -1 sentinel last), add case 8 sorting by `DiskPercent` (float64, with -1 sentinel last, same sentinel-last pattern as IndexingRate)
- [x] Update `sortNodeRows` doc comment column mapping to include cols 7 and 8
- [x] run `make test` — must pass

### Task 5: Verify acceptance criteria

- [x] manual test: `make run ARGS="http://localhost:9200"` — verify Shards and Disk% columns appear in node table, values are populated, "---" displayed when data absent
- [x] run full test suite: `make test`
- [x] run linter: `make lint`

### Task 6: Update documentation

- [x] Update `CLAUDE.md` node table column mapping (currently "7 columns") and key bindings sort hint for node table
- [x] Move this plan to `docs/plans/completed/`
