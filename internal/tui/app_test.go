package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestFetchTimeout(t *testing.T) {
	cases := []struct {
		interval time.Duration
		expected time.Duration
	}{
		{100 * time.Millisecond, 500 * time.Millisecond},  // below min → clamp to 500ms
		{500 * time.Millisecond, 500 * time.Millisecond},  // 0ms after subtraction → min
		{1 * time.Second, 500 * time.Millisecond},         // 500ms → min
		{5 * time.Second, 4500 * time.Millisecond},        // normal
		{10 * time.Second, 9500 * time.Millisecond},       // default interval
		{10500 * time.Millisecond, 10 * time.Second},      // exactly at cap
		{30 * time.Second, 10 * time.Second},              // large interval → capped at 10s
		{300 * time.Second, 10 * time.Second},             // max interval → capped at 10s
	}
	for _, tc := range cases {
		got := fetchTimeout(tc.interval)
		assert.Equal(t, tc.expected, got, "interval=%v", tc.interval)
	}
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

func TestRenderOverview_WidthFillsTerminal(t *testing.T) {
	widths := []int{80, 100, 120, 140, 160, 200}
	for _, w := range widths {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			app := NewApp(nil, 10*time.Second)
			app.width = w

			snap := makeFixtureSnapshot()
			snap.Health.Status = "green"
			snap.Health.NumberOfNodes = 3
			snap.Health.ActiveShards = 10
			app.current = snap
			app.resources = model.ClusterResources{
				AvgCPUPercent:     50.0,
				AvgJVMHeapPercent: 60.0,
				StoragePercent:    70.0,
				StorageUsedBytes:  1024 * 1024 * 1024,
				StorageTotalBytes: 2 * 1024 * 1024 * 1024,
			}

			result := renderOverview(app)
			// Wide mode: the single-line overview row must be exactly app.width wide.
			// lipgloss.Width measures visual column count (ANSI-stripped).
			firstLine := strings.SplitN(result, "\n", 2)[0]
			got := lipgloss.Width(firstLine)
			assert.Equal(t, w, got, "overview first line width should equal terminal width")
		})
	}
}

func TestRenderOverview_UltraNarrowReturnsEmpty(t *testing.T) {
	// Widths 1-5 cannot render two paired cards without content overflow:
	// paired cards get Width(<=2) leaving inner content space of 0 or negative,
	// which causes unbreakable strings (bar chars, metric values) to exceed terminal width.
	// renderOverview must return "" for these widths.
	for _, w := range []int{1, 2, 3, 4, 5} {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			app := NewApp(nil, 10*time.Second)
			app.width = w

			snap := makeFixtureSnapshot()
			snap.Health.Status = "green"
			app.current = snap
			app.resources = model.ClusterResources{}

			assert.Equal(t, "", renderOverview(app), "renderOverview should return empty string for width=%d", w)
		})
	}
}

func TestRenderOverview_NarrowEdgeWidths(t *testing.T) {
	// Narrow mode (< 80) should not overflow at ultra-narrow widths.
	// Width 6 is the minimum: all paired cards have Width(3), inner content space = 1,
	// which allows lipgloss to hard-wrap any single-word content without overflow.
	// Each row is 2 paired cards; their combined width must not exceed app.width.
	narrowWidths := []int{6, 7, 8, 10, 20, 40, 60, 79}
	for _, w := range narrowWidths {
		t.Run(fmt.Sprintf("width=%d", w), func(t *testing.T) {
			app := NewApp(nil, 10*time.Second)
			app.width = w

			snap := makeFixtureSnapshot()
			snap.Health.Status = "green"
			app.current = snap
			app.resources = model.ClusterResources{}

			result := renderOverview(app)
			require.NotEmpty(t, result)

			// Each row in narrow mode must not exceed app.width columns.
			for i, line := range strings.Split(result, "\n") {
				got := lipgloss.Width(line)
				assert.LessOrEqual(t, got, w, "narrow overview line %d width %d > terminal %d", i, got, w)
			}
		})
	}
}

func TestRenderOverview_WideMode_EqualCardHeights(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120

	snap := makeFixtureSnapshot()
	snap.Health.Status = "green"
	snap.Health.NumberOfNodes = 3
	snap.Health.ActiveShards = 10
	app.current = snap
	app.resources = model.ClusterResources{
		AvgCPUPercent:     50.0,
		AvgJVMHeapPercent: 60.0,
		StoragePercent:    70.0,
		StorageUsedBytes:  1024 * 1024 * 1024,
		StorageTotalBytes: 2 * 1024 * 1024 * 1024,
	}

	result := renderOverview(app)
	// Wide mode: JoinHorizontal equalises all cards to maxCardHeight lines.
	// After ANSI stripping, the result must have exactly maxCardHeight lines.
	stripped := stripANSI(result)
	lines := strings.Split(stripped, "\n")
	assert.Equal(t, maxCardHeight, len(lines), "wide mode overview should have exactly maxCardHeight lines")
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

// TestApp_TabSwitchesFocus verifies that Tab toggles activeTable and updates
// focused state on both index and node tables independently.
func TestApp_TabSwitchesFocus(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	// Initial state: activeTable=0, indexTable focused, nodeTable unfocused.
	require.Equal(t, 0, app.activeTable)
	require.True(t, app.indexTable.focused)
	require.False(t, app.nodeTable.focused)

	tab := tea.KeyMsg{Type: tea.KeyTab}

	// First Tab: switch to node table.
	newModel, _ := app.Update(tab)
	updated := newModel.(*App)
	assert.Equal(t, 1, updated.activeTable, "activeTable should be 1 after Tab")
	assert.False(t, updated.indexTable.focused, "indexTable should be unfocused after Tab")
	assert.True(t, updated.nodeTable.focused, "nodeTable should be focused after Tab")

	// Second Tab: switch back to index table.
	newModel, _ = updated.Update(tab)
	updated = newModel.(*App)
	assert.Equal(t, 0, updated.activeTable, "activeTable should wrap back to 0")
	assert.True(t, updated.indexTable.focused, "indexTable should be focused again")
	assert.False(t, updated.nodeTable.focused, "nodeTable should be unfocused again")
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
