# Phase 3: TUI Shell — Header + Overview Bar

## Overview

Build the Bubble Tea application skeleton with auto-refreshing poll loop, header bar, and 7-stat overview bar. After this phase the terminal shows a live-updating colored dashboard header and cluster overview, handling connection errors with auto-retry.

## Context

- Framework: `charmbracelet/bubbletea` + `charmbracelet/lipgloss`
- UI reference: Chrome extension header (PageHeader.tsx) + overview bar (App.tsx lines 279-347)
- Color scheme reference: `src/index.css` CSS variables, Tailwind color classes in components
- Depends on Phase 1 (ES client) and Phase 2 (engine + metrics)

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- TUI components are harder to unit test — focus on testing the logic (state transitions), not the rendered strings
- Test `Update` function with specific messages, verify model fields change correctly

## Testing Strategy

- Test `app.Update()` with `SnapshotMsg` → verify model fields updated
- Test `app.Update()` with `FetchErrorMsg` → verify error state set, consecutive fails counter incremented
- Test `app.Update()` with `tea.WindowSizeMsg` → verify width/height stored
- Test key bindings: `q` → quit command returned, `r` → fetch command returned
- Test `renderOverview()` pure function with fixture data → verify output contains correct values
- Do NOT test lipgloss-rendered color escape codes — too brittle

## Implementation Steps

### Task 1: Add Bubble Tea dependencies

- [ ] `go get github.com/charmbracelet/bubbletea`
- [ ] `go get github.com/charmbracelet/lipgloss`
- [ ] `go get github.com/charmbracelet/bubbles`
- [ ] verify `go mod tidy` succeeds

### Task 2: Custom message types

- [ ] create `internal/tui/messages.go`:
  ```go
  // SnapshotMsg delivers successful poll results to the TUI
  type SnapshotMsg struct {
      Snapshot  *model.Snapshot
      Metrics   model.PerformanceMetrics
      Resources model.ClusterResources
      NodeRows  []model.NodeRow
      IndexRows []model.IndexRow
  }

  // FetchErrorMsg signals a poll failure
  type FetchErrorMsg struct{ Err error }

  // TickMsg triggers the next scheduled poll
  type TickMsg time.Time
  ```

### Task 3: Key bindings

- [ ] create `internal/tui/keys.go`:
  - `q` / `ctrl+c` → quit
  - `r` → force refresh
  - `tab` / `shift+tab` → switch focused table (used in Phase 5)
  - `/` → open search in focused table (used in Phase 5)
  - `esc` → close search / cancel
  - `?` → toggle help text in footer
- [ ] create `helpText` string listing all bindings shown in footer

### Task 4: Lipgloss styles

- [ ] create `internal/tui/styles.go` with all style definitions:
  - Status colors: `colorGreen = lipgloss.Color("#10b981")`, `colorYellow = lipgloss.Color("#f59e0b")`, `colorRed = lipgloss.Color("#ef4444")`, `colorGray = lipgloss.Color("#6b7280")`
  - `StyleStatusGreen`, `StyleStatusYellow`, `StyleStatusRed` — bold + foreground
  - `StyleHeader` — full-width bar, dark background (bg `#1e293b`), white text, padding `0 1`
  - `StyleOverviewCard` — bordered box with gradient-like background per card
  - `StyleMetricCard` — for Phase 4
  - `StyleTableHeader` — bold, underlined, muted foreground
  - `StyleTableRow`, `StyleTableRowAlt` — alternating row colors
  - `StyleError` — red foreground, warning prefix
  - `StyleDim` — muted/gray for secondary info
  - `StyleGreen`, `StyleYellow`, `StyleOrange`, `StyleBlue`, `StylePurple` — for table cell coloring
  - Helper `StatusStyle(status string) lipgloss.Style` returning the right style for "green"/"yellow"/"red"

### Task 5: Root App model

- [ ] create `internal/tui/app.go` with `App` struct:
  ```go
  type App struct {
      client       client.ESClient
      pollInterval time.Duration

      // State
      current   *model.Snapshot
      previous  *model.Snapshot
      metrics   model.PerformanceMetrics
      resources model.ClusterResources
      nodeRows  []model.NodeRow
      indexRows []model.IndexRow
      history   *model.SparklineHistory

      // Connection state
      connState        connState   // connected / disconnected / reconnecting
      consecutiveFails int
      lastError        error
      lastUpdated      time.Time

      // Layout
      width, height int

      // UI state
      showHelp bool
  }

  type connState int
  const (
      stateConnected connState = iota
      stateDisconnected
  )
  ```
- [ ] implement `NewApp(c client.ESClient, interval time.Duration) *App`
- [ ] implement `Init() tea.Cmd` — return `fetchCmd(app.client, nil, app.pollInterval)` (fetch immediately on start)
- [ ] implement `Update(msg tea.Msg) (tea.Model, tea.Cmd)`:
  - `tea.WindowSizeMsg` → store width/height
  - `SnapshotMsg` → rotate previous/current, store metrics/resources/rows, push sparkline point, reset fails, return `tickCmd`
  - `FetchErrorMsg` → increment fails, store error, compute backoff (min(2^fails * time.Second, 60s)), return `tea.Tick(backoff, ...)`
  - `TickMsg` → return `fetchCmd`
  - `tea.KeyMsg` → handle q/r/tab/? keys
- [ ] implement `View() string` — calls `renderHeader() + renderOverview() + renderFooter()` joined vertically; content area placeholder for Phase 4/5
- [ ] implement `tickCmd(d time.Duration) tea.Cmd` and `fetchCmd(c, prev, interval) tea.Cmd`
  - `fetchCmd` calls `engine.FetchAll`, then `engine.CalcClusterMetrics/Resources/NodeRows/IndexRows`, returns `SnapshotMsg` or `FetchErrorMsg`

### Task 6: Header bar renderer

- [ ] create `internal/tui/header.go` with `renderHeader(app *App) string`:
  - left: cluster name (or "Connecting..." if no snapshot yet)
  - center: status indicator — colored dot + status text ("● GREEN", "● YELLOW", "● RED", "● UNKNOWN")
    - "DISCONNECTED" + error hint in red if `connState == stateDisconnected`
  - right: "Last: HH:MM:SS  Poll: Ns" (or "Connecting..." during first fetch)
  - full terminal width via `lipgloss.Width(app.width)`
  - dark background, white text
- [ ] handle case: no snapshot yet (app.current == nil) → show "Connecting to <baseURL>..."
- [ ] handle "DISCONNECTED" state: show error in red, "Press r to retry"

### Task 7: Overview bar renderer

- [ ] create `internal/tui/overview.go` with `renderOverview(app *App) string`:
  - returns empty string if `app.current == nil`
  - 7 cards in a horizontal row, equal width = `(app.width - 6) / 7`
  - Card 1: **Status** — colored background, large text "GREEN"/"YELLOW"/"RED"
  - Card 2: **Nodes** — count + "Nodes" label, blue foreground
  - Card 3: **Indices** — count + "Indices" label, purple foreground
  - Card 4: **Shards** — `active_shards` + "Active Shards", indigo foreground
  - Card 5: **CPU** — `AvgCPUPercent` + "%" + mini progress bar
    - mini bar: `[████░░░░░░]` using block characters, width ~10
    - yellow text if > 80%, red if > 90%
  - Card 6: **JVM** — `AvgJVMHeapPercent` + mini progress bar
    - yellow if > 75%, red if > 85%
  - Card 7: **Storage** — `StoragePercent` + "used/total GB" below
    - yellow if > 80%, red if > 90%
  - join all 7 cards with `lipgloss.JoinHorizontal(lipgloss.Top, cards...)`
- [ ] implement `renderMiniBar(percent float64, width int) string` helper — fills proportionally with "█" and "░"

### Task 8: Footer renderer

- [ ] create footer within `view.go` or header.go:
  - show `helpText` when `app.showHelp` is true, otherwise show brief "? for help"
  - dim/muted color
  - full width

### Task 9: Wire into main.go

- [ ] update `cmd/epm/main.go` to create `tui.NewApp(client, interval)` and run `tea.NewProgram(app, tea.WithAltScreen())`
- [ ] remove debug print output from Phase 2
- [ ] verify app launches, shows header with cluster status, auto-refreshes

### Task 10: Tests for app logic

- [ ] create `internal/tui/app_test.go`
- [ ] `TestApp_SnapshotMsgUpdatesState` — send SnapshotMsg, verify `current` set, `consecutiveFails` reset
- [ ] `TestApp_FetchErrorIncreasesFails` — send FetchErrorMsg twice, verify `consecutiveFails == 2`
- [ ] `TestApp_WindowSizeStored` — send WindowSizeMsg, verify width/height
- [ ] `TestApp_QuitKey` — send KeyMsg "q", verify quit command returned
- [ ] `TestRenderMiniBar` — verify correct fill ratio
- [ ] run `go test ./internal/tui/...` — all pass

### Task 11: Final verification

- [ ] run `go test ./...` — all pass
- [ ] run `go vet ./...` — no issues
- [ ] launch against live ES cluster: header shows cluster name, status color is correct, refreshes on schedule
- [ ] disconnect ES: shows "DISCONNECTED" error, retries with backoff, recovers when ES comes back

## Technical Details

- `tea.WithAltScreen()` ensures the TUI uses the alternate terminal buffer and restores on exit
- `fetchCmd` creates its own `context.WithTimeout` — slightly less than poll interval to avoid overlap
- Backoff formula: `min(2^consecutiveFails * time.Second, 60 * time.Second)` — starts at 2s, caps at 60s
- Mini progress bar uses Unicode block chars: `█` (U+2588) for filled, `░` (U+2591) for empty

## Post-Completion

- Verify terminal reset is clean on exit (no leftover escape codes)
- Test at narrow terminal width (80 cols) and wide (200 cols)
