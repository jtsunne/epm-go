package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dm/epm-go/internal/model"
)

// makeFixtureSnapshot returns a minimal Snapshot for testing.
func makeFixtureSnapshot() *model.Snapshot {
	return &model.Snapshot{
		FetchedAt: time.Now(),
	}
}

// makeFixtureMsg builds a SnapshotMsg with the given snapshot.
func makeFixtureMsg(snap *model.Snapshot) SnapshotMsg {
	return SnapshotMsg{
		Snapshot: snap,
		Metrics: model.PerformanceMetrics{
			IndexingRate: 100,
			SearchRate:   200,
		},
		Resources: model.ClusterResources{
			AvgCPUPercent: 42,
		},
		NodeRows:  []model.NodeRow{{Name: "node-1"}},
		IndexRows: []model.IndexRow{{Name: "my-index"}},
	}
}

func TestApp_SnapshotMsgUpdatesState(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	require.Nil(t, app.current)
	require.Equal(t, 0, app.consecutiveFails)

	snap := makeFixtureSnapshot()
	msg := makeFixtureMsg(snap)

	newModel, cmd := app.Update(msg)
	updated := newModel.(*App)

	assert.Equal(t, snap, updated.current)
	assert.Nil(t, updated.previous)
	assert.False(t, updated.fetching)
	assert.Equal(t, 0, updated.consecutiveFails)
	assert.Nil(t, updated.lastError)
	assert.Equal(t, stateConnected, updated.connState)
	assert.Equal(t, msg.Metrics, updated.metrics)
	assert.Equal(t, msg.Resources, updated.resources)
	assert.Equal(t, msg.NodeRows, updated.nodeRows)
	assert.Equal(t, msg.IndexRows, updated.indexRows)
	assert.Equal(t, snap.FetchedAt, updated.lastUpdated)
	// First poll has no previous snapshot, so no history point is recorded.
	assert.Equal(t, 0, updated.history.Len())
	require.NotNil(t, cmd)
}

func TestApp_SnapshotMsgRotatesPreviousCurrent(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	snap1 := makeFixtureSnapshot()
	snap2 := makeFixtureSnapshot()

	// First snapshot
	newModel, _ := app.Update(makeFixtureMsg(snap1))
	app = newModel.(*App)

	// Second snapshot — snap1 becomes previous
	newModel, _ = app.Update(makeFixtureMsg(snap2))
	app = newModel.(*App)

	assert.Equal(t, snap2, app.current)
	assert.Equal(t, snap1, app.previous)
}

func TestApp_FetchErrorIncreasesFails(t *testing.T) {
	app := NewApp(nil, 10*time.Second)

	err1 := errors.New("connection refused")
	newModel, cmd1 := app.Update(FetchErrorMsg{Err: err1})
	app = newModel.(*App)

	assert.Equal(t, 1, app.consecutiveFails)
	assert.Equal(t, err1, app.lastError)
	assert.Equal(t, stateDisconnected, app.connState)
	require.NotNil(t, cmd1)

	newModel, cmd2 := app.Update(FetchErrorMsg{Err: err1})
	app = newModel.(*App)

	assert.Equal(t, 2, app.consecutiveFails)
	require.NotNil(t, cmd2)
}

func TestApp_FetchErrorResetsOnSuccess(t *testing.T) {
	app := NewApp(nil, 10*time.Second)

	// Simulate two failures
	newModel, _ := app.Update(FetchErrorMsg{Err: errors.New("timeout")})
	newModel, _ = newModel.(*App).Update(FetchErrorMsg{Err: errors.New("timeout")})
	app = newModel.(*App)
	require.Equal(t, 2, app.consecutiveFails)

	// Now a successful snapshot resets the counter
	snap := makeFixtureSnapshot()
	newModel, _ = app.Update(makeFixtureMsg(snap))
	app = newModel.(*App)

	assert.Equal(t, 0, app.consecutiveFails)
	assert.Nil(t, app.lastError)
	assert.Equal(t, stateConnected, app.connState)
}

func TestApp_WindowSizeStored(t *testing.T) {
	app := NewApp(nil, 10*time.Second)

	newModel, cmd := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := newModel.(*App)

	assert.Equal(t, 120, updated.width)
	assert.Equal(t, 40, updated.height)
	assert.Nil(t, cmd)
}

func TestApp_QuitKey(t *testing.T) {
	app := NewApp(nil, 10*time.Second)

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	_ = newModel

	// tea.Quit is a function — we verify a non-nil command is returned.
	require.NotNil(t, cmd)
	// Execute the command; it should return tea.QuitMsg.
	result := cmd()
	_, isQuit := result.(tea.QuitMsg)
	assert.True(t, isQuit, "expected tea.QuitMsg, got %T", result)
}

func TestApp_RefreshKey(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.fetching = false

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	updated := newModel.(*App)

	require.NotNil(t, cmd, "expected fetch command returned for 'r' key")
	assert.True(t, updated.fetching)
}

func TestApp_RefreshKeyNoopWhileFetching(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.fetching = true

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	assert.Nil(t, cmd)
}

func TestApp_HelpToggle(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	require.False(t, app.showHelp)

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	app = newModel.(*App)
	assert.True(t, app.showHelp)

	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	app = newModel.(*App)
	assert.False(t, app.showHelp)
}

func TestBackoffDuration(t *testing.T) {
	cases := []struct {
		fails    int
		expected time.Duration
	}{
		{0, time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 32 * time.Second},
		{6, 60 * time.Second},
		{10, 60 * time.Second},
	}
	for _, tc := range cases {
		got := backoffDuration(tc.fails)
		assert.Equal(t, tc.expected, got, "fails=%d", tc.fails)
	}
}

func TestRenderMiniBar(t *testing.T) {
	cases := []struct {
		percent  float64
		width    int
		wantFill int
	}{
		{0, 10, 0},
		{100, 10, 10},
		{50, 10, 5},
		{25, 8, 2},
		{75, 8, 6},
	}
	for _, tc := range cases {
		result := renderMiniBar(tc.percent, tc.width)
		assert.Len(t, []rune(result), tc.width, "total bar width percent=%v", tc.percent)
		filledCount := strings.Count(result, "█")
		assert.Equal(t, tc.wantFill, filledCount, "filled count percent=%v width=%v", tc.percent, tc.width)
	}
	// Zero width returns empty string.
	assert.Equal(t, "", renderMiniBar(50, 0))
}

func TestApp_SparklineNonEmptyAfterThreePolls(t *testing.T) {
	app := NewApp(nil, 10*time.Second)

	// Push 3 snapshots with non-zero metrics.
	for i := 1; i <= 3; i++ {
		snap := makeFixtureSnapshot()
		msg := SnapshotMsg{
			Snapshot: snap,
			Metrics: model.PerformanceMetrics{
				IndexingRate: float64(i * 100),
				SearchRate:   float64(i * 50),
			},
		}
		newModel, _ := app.Update(msg)
		app = newModel.(*App)
	}

	// First poll is skipped (no previous snapshot), so 3 polls yield 2 history points.
	require.Equal(t, 2, app.history.Len())

	values := app.history.Values("indexingRate")
	require.Len(t, values, 2)

	sparkline := stripANSI(RenderSparkline(values, 10, testColor))
	assert.NotEqual(t, strings.Repeat(" ", 10), sparkline, "sparkline should contain non-space chars after 3 polls")
	// With 3 values and width 10, the right side contains sparkline chars (left-padded with spaces).
	assert.Contains(t, sparkline, "█", "sparkline should contain a max-value char")
}

func TestRenderOverview_NilSnapshot(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120
	assert.Equal(t, "", renderOverview(app))
}

func TestRenderOverview_WithSnapshot(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120

	snap := makeFixtureSnapshot()
	snap.Health.Status = "green"
	snap.Health.NumberOfNodes = 5
	snap.Health.ActiveShards = 42

	app.current = snap
	app.resources = model.ClusterResources{
		AvgCPUPercent:     34.5,
		AvgJVMHeapPercent: 67.2,
		StoragePercent:    45.0,
		StorageUsedBytes:  512 * 1024 * 1024,
		StorageTotalBytes: 1024 * 1024 * 1024,
	}

	result := renderOverview(app)
	assert.NotEmpty(t, result)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "GREEN")
	assert.Contains(t, stripped, "5")
	assert.Contains(t, stripped, "42")
}

// stripANSI removes ANSI escape sequences for plain-text content assertions.
// Handles all CSI sequences (not just SGR m-terminated ones).
func stripANSI(s string) string {
	var out strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			// CSI final bytes are in range 0x40–0x7E (@, A-Z, [, \, ], ^, _, `, a-z, {, |, }, ~)
			if r >= 0x40 && r <= 0x7E {
				inEscape = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
