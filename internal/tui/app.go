package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

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

	// Layout
	width, height int

	// UI state
	showHelp bool
}

// NewApp creates a new App with the given ES client and poll interval.
func NewApp(c client.ESClient, interval time.Duration) *App {
	return &App{
		client:       c,
		pollInterval: interval,
		history:      model.NewSparklineHistory(0),
		connState:    stateDisconnected,
		fetching:     true, // Init() always issues an immediate fetchCmd
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

	case SnapshotMsg:
		app.fetching = false
		app.previous = app.current
		app.current = msg.Snapshot
		app.metrics = msg.Metrics
		app.resources = msg.Resources
		app.nodeRows = msg.NodeRows
		app.indexRows = msg.IndexRows
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
		return app, tickCmd(app.pollInterval)

	case FetchErrorMsg:
		app.fetching = false
		app.consecutiveFails++
		app.lastError = msg.Err
		app.connState = stateDisconnected
		backoff := backoffDuration(app.consecutiveFails)
		return app, tea.Tick(backoff, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})

	case TickMsg:
		if app.fetching {
			return app, nil
		}
		app.fetching = true
		return app, fetchCmd(app.client, app.current, app.pollInterval)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return app, tea.Quit
		case key.Matches(msg, keys.Refresh):
			if app.fetching {
				return app, nil
			}
			app.fetching = true
			return app, fetchCmd(app.client, app.current, app.pollInterval)
		case key.Matches(msg, keys.Help):
			app.showHelp = !app.showHelp
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
	parts = append(parts, renderFooter(app))

	return strings.Join(parts, "\n")
}

// tickCmd schedules the next poll after duration d.
func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// fetchCmd is a Bubble Tea command that calls all 5 ES endpoints, computes
// derived metrics, and returns a SnapshotMsg or FetchErrorMsg.
func fetchCmd(c client.ESClient, prev *model.Snapshot, interval time.Duration) tea.Cmd {
	return func() tea.Msg {
		timeout := interval - 500*time.Millisecond
		if timeout < 500*time.Millisecond {
			timeout = 500 * time.Millisecond
		}
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
