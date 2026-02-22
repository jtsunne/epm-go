package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dm/epm-go/internal/client"
	"github.com/dm/epm-go/internal/engine"
	"github.com/dm/epm-go/internal/model"
)

type connState int

const (
	stateConnected    connState = iota
	stateDisconnected connState = iota
)

// App is the root Bubble Tea model for epm.
type App struct {
	client       client.ESClient
	pollInterval time.Duration

	// Poll state
	fetching  bool // true while a fetchCmd goroutine is in-flight
	tickGen   int  // incremented each time a new tick is scheduled; stale ticks are dropped
	current   *model.Snapshot
	previous  *model.Snapshot
	metrics   model.PerformanceMetrics
	resources model.ClusterResources
	nodeRows  []model.NodeRow
	indexRows []model.IndexRow
	history   *model.SparklineHistory

	// Connection state
	connState        connState
	consecutiveFails int
	lastError        error
	lastUpdated      time.Time
	nextRetryAt      time.Time // when the next backoff retry is scheduled
	countdownGen     int       // incremented to invalidate stale CountdownTickMsgs

	// Layout
	width, height int

	// UI state
	showHelp bool

	// Tables
	indexTable  IndexTableModel
	nodeTable   NodeTableModel
	activeTable int // 0 = index table, 1 = node table
}

// NewApp creates a new App with the given ES client and poll interval.
func NewApp(c client.ESClient, interval time.Duration) *App {
	it := NewIndexTable()
	it.focused = true // index table is focused by default
	nt := NewNodeTable()
	return &App{
		client:       c,
		pollInterval: interval,
		history:      model.NewSparklineHistory(60),
		connState:    stateDisconnected,
		fetching:     true, // Init() always issues an immediate fetchCmd
		indexTable:   it,
		nodeTable:    nt,
		activeTable:  0,
	}
}

// Init implements tea.Model. Starts the first fetch immediately on launch.
func (app *App) Init() tea.Cmd {
	return fetchCmd(app.client, nil, app.pollInterval)
}

// Update implements tea.Model — the single state-mutation entry point.
func (app *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		app.width = msg.Width
		app.height = msg.Height
		app.computeTablePageSizes()

	case SnapshotMsg:
		app.fetching = false
		app.previous = app.current
		app.current = msg.Snapshot
		app.metrics = msg.Metrics
		app.resources = msg.Resources
		app.nodeRows = msg.NodeRows
		app.indexRows = msg.IndexRows
		app.indexTable.SetData(msg.IndexRows)
		app.nodeTable.SetData(msg.NodeRows)
		app.computeTablePageSizes()
		// Only push to history when we have a previous snapshot — the first
		// poll has no delta, so all rate/latency metrics are zero and would
		// corrupt the sparkline baseline.
		if app.previous != nil {
			app.history.Push(model.SparklinePoint{
				Timestamp:     msg.Snapshot.FetchedAt,
				IndexingRate:  msg.Metrics.IndexingRate,
				SearchRate:    msg.Metrics.SearchRate,
				IndexLatency:  msg.Metrics.IndexLatency,
				SearchLatency: msg.Metrics.SearchLatency,
			})
		}
		app.consecutiveFails = 0
		app.lastError = nil
		app.connState = stateConnected
		app.lastUpdated = msg.Snapshot.FetchedAt
		app.nextRetryAt = time.Time{}
		app.countdownGen++ // invalidate any pending countdown tick
		app.tickGen++
		return app, tickCmd(app.pollInterval, app.tickGen)

	case FetchErrorMsg:
		app.fetching = false
		app.consecutiveFails++
		app.lastError = msg.Err
		app.connState = stateDisconnected
		delay := backoffDuration(app.consecutiveFails)
		app.nextRetryAt = time.Now().Add(delay)
		app.tickGen++
		app.countdownGen++
		return app, tea.Batch(
			tickCmd(delay, app.tickGen),
			countdownTickCmd(time.Second, app.countdownGen),
		)

	case CountdownTickMsg:
		if msg.Gen != app.countdownGen {
			return app, nil
		}
		if app.connState == stateConnected {
			return app, nil
		}
		return app, countdownTickCmd(time.Second, app.countdownGen)

	case TickMsg:
		if msg.Gen != app.tickGen {
			return app, nil
		}
		if app.fetching {
			return app, nil
		}
		app.fetching = true
		return app, fetchCmd(app.client, app.current, app.pollInterval)

	case tea.KeyMsg:
		// ctrl+c / q always quit, even during table search.
		if key.Matches(msg, keys.Quit) {
			return app, tea.Quit
		}

		// While the active table has its search input open, delegate all
		// other keys to the table so typed characters reach the text field.
		activeSearching := (app.activeTable == 0 && app.indexTable.searching) ||
			(app.activeTable == 1 && app.nodeTable.searching)
		if activeSearching {
			var cmd tea.Cmd
			if app.activeTable == 0 {
				app.indexTable, cmd = app.indexTable.Update(msg)
			} else {
				app.nodeTable, cmd = app.nodeTable.Update(msg)
			}
			return app, cmd
		}

		switch {
		case key.Matches(msg, keys.Refresh):
			if app.fetching {
				return app, nil
			}
			app.tickGen++ // invalidate any pending tick so it doesn't trigger a double-fetch
			app.fetching = true
			return app, fetchCmd(app.client, app.current, app.pollInterval)
		case key.Matches(msg, keys.Tab), key.Matches(msg, keys.ShiftTab):
			app.activeTable = (app.activeTable + 1) % 2
			app.indexTable.focused = app.activeTable == 0
			app.nodeTable.focused = app.activeTable == 1
		case key.Matches(msg, keys.Help):
			app.showHelp = !app.showHelp
		default:
			var cmd tea.Cmd
			if app.activeTable == 0 {
				app.indexTable, cmd = app.indexTable.Update(msg)
			} else {
				app.nodeTable, cmd = app.nodeTable.Update(msg)
			}
			return app, cmd
		}
	}

	return app, nil
}

// View implements tea.Model. Renders the full TUI.
func (app *App) View() string {
	var parts []string

	if h := renderHeader(app); h != "" {
		parts = append(parts, h)
	}
	if o := renderOverview(app); o != "" {
		parts = append(parts, o)
	}
	if m := renderMetricsRow(app); m != "" {
		parts = append(parts, m)
	}
	// Compact mode: terminal height < 40 — show only the active table to
	// maximise the visible rows on small screens.
	if app.height > 0 && app.height < 40 {
		if app.activeTable == 0 {
			parts = append(parts, app.indexTable.renderTable(app))
		} else {
			parts = append(parts, app.nodeTable.renderTable(app))
		}
	} else {
		parts = append(parts, app.indexTable.renderTable(app))
		parts = append(parts, app.nodeTable.renderTable(app))
	}
	parts = append(parts, renderFooter(app))

	return strings.Join(parts, "\n")
}

// tickCmd schedules the next poll after duration d, embedding gen so the
// TickMsg handler can discard ticks that belong to a superseded schedule.
func tickCmd(d time.Duration, gen int) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickMsg{Time: t, Gen: gen}
	})
}

// countdownTickCmd schedules a CountdownTickMsg after duration d with the given
// gen so the header countdown display updates every second while disconnected.
func countdownTickCmd(d time.Duration, gen int) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return CountdownTickMsg{Gen: gen}
	})
}

// fetchTimeout returns the context timeout for a fetch command.
// It is interval - 500ms, clamped to [500ms, 10s].
// The 10s upper bound ensures that any in-flight fetch cancels promptly
// when the user quits, regardless of how large the poll interval is.
func fetchTimeout(interval time.Duration) time.Duration {
	const (
		minTimeout = 500 * time.Millisecond
		maxTimeout = 10 * time.Second
	)
	t := interval - 500*time.Millisecond
	if t < minTimeout {
		t = minTimeout
	}
	if t > maxTimeout {
		t = maxTimeout
	}
	return t
}

// fetchCmd is a Bubble Tea command that calls all 5 ES endpoints, computes
// derived metrics, and returns a SnapshotMsg or FetchErrorMsg.
//
// The fetch context timeout is capped at 10s so that any in-flight HTTP
// request is always cancelled promptly when the user quits — Bubble Tea
// stops dispatching messages immediately on tea.Quit, but goroutines
// spawned by commands run until their own contexts expire.  The errgroup
// in FetchAll propagates cancellation to all 5 concurrent goroutines, and
// the HTTP client respects context cancellation via NewRequestWithContext.
func fetchCmd(c client.ESClient, prev *model.Snapshot, interval time.Duration) tea.Cmd {
	return func() tea.Msg {
		timeout := fetchTimeout(interval)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		snap, err := engine.FetchAll(ctx, c)
		if err != nil {
			return FetchErrorMsg{Err: err}
		}

		var elapsed time.Duration
		if prev != nil {
			elapsed = snap.FetchedAt.Sub(prev.FetchedAt)
		}

		metrics := engine.CalcClusterMetrics(prev, snap, elapsed)
		resources := engine.CalcClusterResources(snap)
		nodeRows := engine.CalcNodeRows(prev, snap, elapsed)
		indexRows := engine.CalcIndexRows(prev, snap, elapsed)

		return SnapshotMsg{
			Snapshot:  snap,
			Metrics:   metrics,
			Resources: resources,
			NodeRows:  nodeRows,
			IndexRows: indexRows,
		}
	}
}

// computeTablePageSizes updates the pageSize of both tables to fill the
// available terminal height after the fixed UI sections (header, overview,
// metrics, footer) are accounted for.
//
// In compact mode (height < 40), only the active table is visible and receives
// all the available vertical space. In normal mode the available space is split
// 60 % to the index table and 40 % to the node table.
func (app *App) computeTablePageSizes() {
	if app.height <= 0 {
		return
	}

	// Measure rendered heights of the non-table sections.
	// renderedHeight returns 0 for empty strings (overview and metrics return
	// empty before the first successful fetch).
	fixedH := renderedHeight(renderHeader(app)) +
		renderedHeight(renderOverview(app)) +
		renderedHeight(renderMetricsRow(app)) +
		renderedHeight(renderFooter(app))

	// Each rendered table section costs:
	//   1 title bar line  +  1 column-header row  +  1 separator line = 3 overhead lines.
	// The remaining lines are available for data rows (pageSize).
	const tableOverhead = 3

	totalAvailable := app.height - fixedH

	if app.height < 40 {
		// Compact mode: a single table fills the remaining space.
		rows := totalAvailable - tableOverhead
		if rows < 1 {
			rows = 1
		}
		app.indexTable.pageSize = rows
		app.nodeTable.pageSize = rows
	} else {
		// Normal mode: 60 % to index table, 40 % to node table.
		idxH := totalAvailable * 60 / 100
		nodeH := totalAvailable - idxH

		idxRows := idxH - tableOverhead
		nodeRows := nodeH - tableOverhead
		if idxRows < 1 {
			idxRows = 1
		}
		if nodeRows < 1 {
			nodeRows = 1
		}
		app.indexTable.pageSize = idxRows
		app.nodeTable.pageSize = nodeRows
	}

	// Clamp pages so they stay in range after a resize.
	app.indexTable.clampPage(len(app.indexTable.displayRows))
	app.nodeTable.clampPage(len(app.nodeTable.displayRows))
}

// LastError returns the most recent fetch error, or nil if the last fetch
// was successful. Used by main to decide whether to print a post-exit hint.
func (app *App) LastError() error {
	return app.lastError
}

// renderedHeight returns the number of terminal lines a rendered section
// occupies. Returns 0 for empty strings (sections absent before data arrives).
func renderedHeight(s string) int {
	if s == "" {
		return 0
	}
	return lipgloss.Height(s)
}

// backoffDuration returns min(2^fails * time.Second, 60*time.Second).
// At fails=1: 2s, fails=2: 4s, fails=3: 8s, ..., fails>=6: 60s.
func backoffDuration(fails int) time.Duration {
	const maxBackoff = 60 * time.Second
	if fails <= 0 {
		return time.Second
	}
	if fails >= 6 {
		return maxBackoff
	}
	return time.Duration(1<<fails) * time.Second
}
