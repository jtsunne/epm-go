# Allow Adjusting Index Settings via TUI Form

## Overview
Add an index settings editor to the TUI: pressing `e` (edit) on the index table opens a full-screen form overlay where the user can view and modify common dynamic Elasticsearch index settings for one or multiple selected indices. The form pre-populates current values from the ES `_settings` API, shows per-field suggestions (including live node names/IPs for routing allocation fields), and sends a PUT request with only the changed fields on confirmation.

## Context
- Files involved:
  - Modify: `internal/client/types.go`
  - Modify: `internal/client/client.go` (ESClient interface)
  - Modify: `internal/client/endpoints.go` (GetIndexSettings, UpdateIndexSettings impl)
  - Modify: `internal/client/client_test.go`
  - Modify: `internal/engine/mock_client_test.go`
  - Create: `internal/tui/settings.go` (SettingsFormModel, settingsLoadCmd, settingsUpdateCmd, renderSettingsForm)
  - Create: `internal/tui/settings_test.go`
  - Modify: `internal/tui/messages.go` (SettingsLoadedMsg, SettingsResultMsg)
  - Modify: `internal/tui/keys.go` (EditSettings key `e`)
  - Modify: `internal/tui/app.go` (state fields, message handlers, View routing)
- Related patterns: deleteConfirmMode/renderDeleteConfirm, deleteCmd/DeleteResultMsg, tableModel textinput usage
- Dependencies: `charmbracelet/bubbles/textinput` (already in go.mod)

## Settings Exposed (11 fields)
1. `index.number_of_replicas` — integer; suggestions: 0, 1, 2, 3
2. `index.refresh_interval` — text; suggestions: -1, 1s, 5s, 30s, 60s
3. `index.routing.allocation.include._name` — text; suggestions: node names from current snapshot
4. `index.routing.allocation.exclude._name` — text; suggestions: node names
5. `index.routing.allocation.require._name` — text; suggestions: node names
6. `index.routing.allocation.include._ip` — text; suggestions: node IPs from snapshot
7. `index.routing.allocation.exclude._ip` — text; suggestions: node IPs
8. `index.routing.allocation.require._ip` — text; suggestions: node IPs
9. `index.routing.allocation.total_shards_per_node` — integer; suggestions: -1, 1, 2, 5
10. `index.mapping.total_fields.limit` — integer; suggestions: 1000, 2000, 5000, 10000
11. `index.blocks.read_only_allow_delete` — bool text; suggestions: true, false, "" (clear)

## UX Flow
- `e` pressed on index table → collect selected names (or cursor row) → issue settingsLoadCmd (GET settings for first index) → enter settingsMode with "Loading..." state
- SettingsLoadedMsg arrives → form fields pre-populated with current values; node names/IPs extracted from existing snapshot (no extra API call)
- Navigate fields with ↑/↓; type to edit the focused field (textinput per field, focused one shows cursor)
- `ctrl+s` → submit only changed fields via PUT `/<index1,index2>/_settings`
- `esc` → cancel, return to dashboard
- SettingsResultMsg arrives → show success/error status in footer (same pattern as deleteStatus), refresh data

## Development Approach
- Testing approach: Regular (code first, then tests)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Client layer — new types and methods

**Files:**
- Modify: `internal/client/types.go`
- Modify: `internal/client/client.go`
- Modify: `internal/client/endpoints.go`
- Modify: `internal/client/client_test.go`
- Modify: `internal/engine/mock_client_test.go`

- [x] Add to `types.go`: `IndexSettingsValues`, `IndexRoutingSettings`, `IndexAllocationSettings`, `IndexAllocationFilter`, `IndexMappingSettings`, `IndexBlocksSettings` (JSON structs matching GET `/_settings` response shape)
- [x] Add to `client.go` ESClient interface: `GetIndexSettings(ctx context.Context, name string) (*IndexSettingsValues, error)` and `UpdateIndexSettings(ctx context.Context, names []string, settings map[string]any) error`
- [x] Add `doPutJSON(ctx context.Context, path string, body []byte) error` to DefaultClient (like doDelete but PUT with JSON body and Content-Type header)
- [x] Add to `endpoints.go`: `GetIndexSettings` — `GET /<name>/_settings?filter_path=*.settings.index.number_of_replicas,*.settings.index.refresh_interval,*.settings.index.routing.allocation.*,*.settings.index.mapping.total_fields.limit,*.settings.index.blocks.read_only_allow_delete` — unmarshal to map then extract first entry's settings.index
- [x] Add to `endpoints.go`: `UpdateIndexSettings` — build nested map from flat dotted keys (e.g. `"index.number_of_replicas": 2`), marshal to JSON, call `doPutJSON` to `/<names,joined>/_settings`
- [x] Add httptest tests for `GetIndexSettings` (verify path, filter_path, JSON decoding) and `UpdateIndexSettings` (verify PUT method, path, body content)
- [x] Update `MockESClient` in `internal/engine/mock_client_test.go`: add `GetIndexSettingsFn` and `UpdateIndexSettingsFn` fields + stub methods matching the new interface methods
- [x] run `make test` — must pass before task 2

### Task 2: Settings form model

**Files:**
- Create: `internal/tui/settings.go`
- Create: `internal/tui/settings_test.go`

- [x] Define `settingsField` struct: Label, ESKey (full dotted key e.g. `index.number_of_replicas`), currentVal string, suggestions []string, input textinput.Model
- [x] Define `SettingsFormModel` struct: fields []settingsField, focusedField int, loading bool, loadErr string, names []string (index names being edited)
- [x] Implement `buildSettingsForm(names []string, nodeNames []string, nodeIPs []string) SettingsFormModel` — creates all 11 fields with empty current values and suggestions populated from nodeNames/nodeIPs; focuses first field
- [x] Implement `(m *SettingsFormModel) applySettings(vals *client.IndexSettingsValues)` — populates currentVal and pre-fills each input with current value from ES
- [x] Implement `(m SettingsFormModel) Update(msg tea.Msg) (SettingsFormModel, tea.Cmd)` — routes key to focused field's textinput; ↑/↓ (and Tab/Shift+Tab) changes focusedField (blur old, focus new); ctrl+s returns a `submitSettingsMsg{}` sentinel; esc returns a `cancelSettingsMsg{}` sentinel (these are unexported local types signaling intent, not tea.Msgs)
- [x] Implement `(m *SettingsFormModel) changedSettings() map[string]any` — returns map of ESKey → new value for all fields where input.Value() != currentVal, skipping identical values; empty string value is included (to clear a setting)
- [x] Write table-driven tests in `settings_test.go`:
  - `buildSettingsForm` populates correct number of fields (11) and routing suggestions
  - `changedSettings` returns empty map when no changes; returns correct changed fields when values differ; excludes unchanged fields
  - Navigation: Update with ↑/↓ keys changes focusedField correctly; wraps at boundaries
- [x] run `make test` — must pass before task 3

### Task 3: App integration

**Files:**
- Modify: `internal/tui/messages.go`
- Modify: `internal/tui/keys.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/settings.go` (add settingsLoadCmd, settingsUpdateCmd, renderSettingsForm)

- [x] Add to `messages.go`: `SettingsLoadedMsg{Values *client.IndexSettingsValues, Err error}` and `SettingsResultMsg{Names []string, Err error}`
- [x] Add to `keys.go`: `EditSettings key.Binding` bound to `"e"` with help `"e: edit settings"`; add to helpText constant
- [x] Add to `app.go` App struct: `settingsMode bool`, `settingsForm SettingsFormModel`, `settingsStatus string`, `settingsStatusErr bool`
- [x] Add `settingsLoadCmd(c client.ESClient, name string) tea.Cmd` to `settings.go` — calls `GetIndexSettings` and returns `SettingsLoadedMsg`
- [x] Add `settingsUpdateCmd(c client.ESClient, names []string, settings map[string]any) tea.Cmd` to `settings.go` — calls `UpdateIndexSettings` and returns `SettingsResultMsg`
- [x] In `app.go Update()`, handle `SettingsLoadedMsg`: call `m.settingsForm.applySettings(msg.Values)` (or set loadErr); mark form loading=false
- [x] In `app.go Update()`, handle `SettingsResultMsg`: exit settingsMode; set settingsStatus success/error (cleared on next SnapshotMsg); if success and not already fetching, trigger immediate refresh (same pattern as delete)
- [x] In `app.go Update()`, handle `tea.KeyMsg` in settingsMode:
  - Route to `settingsForm.Update(msg)` for all keys
  - Detect `ctrl+s` result: extract changed settings, issue settingsUpdateCmd if any changes exist (or exit with no-op if empty)
  - Detect `esc` result: set settingsMode=false
- [x] In `app.go Update()`, handle `tea.KeyMsg` in normal mode: add `case key.Matches(msg, keys.EditSettings) && app.activeTable == 0:` — collect names (same logic as delete), enter settingsMode, issue settingsLoadCmd, extract nodeNames/nodeIPs from current snapshot NodeStats
- [x] Clear `settingsStatus` on `SnapshotMsg` and `FetchErrorMsg` (same as deleteStatus)
- [x] In `app.go View()`: add `if app.settingsMode { parts = append(parts, renderSettingsForm(app)); ... return }` before the dashboard, similar to deleteConfirmMode
- [x] Add `renderSettingsForm(app *App) string` to `settings.go`:
  - Title bar styled like `renderDeleteConfirm`: "Edit Index Settings: <names>" + "[ctrl+s: save  esc: cancel]"
  - For each field: one or two lines — label+current value on left, input on right, suggestions dimmed below
  - Focused field row highlighted with `colorSelectedBg`
  - Loading state shows "Loading current settings..." centered in the form
  - LoadErr state shows error message
- [x] run `make test` — must pass before task 4

### Task 4: Verify acceptance criteria

- [x] manual test: press `e` on single index → form opens with current values → edit replicas → ctrl+s → success status shown → index table refreshes
- [x] manual test: select 2 indices with space, press `e` → form shows first index settings, title says "2 indices" → save applies to both
- [x] manual test: press `esc` in form → returns to dashboard, no changes made
- [x] manual test: routing allocation fields show available node names/IPs as suggestions
- [x] run `make lint` — must pass with no new warnings
- [x] run `make test` — all tests must pass
- [x] run `make build` — binary must compile cleanly

### Task 5: Update documentation

- [x] Update CLAUDE.md: add `settingsMode` to Architecture Notes section (alongside deleteConfirmMode); add `EditSettings` key `e` to keyboard shortcuts table; add `settings.go` to project structure
- [x] Move this plan to `docs/plans/completed/`
