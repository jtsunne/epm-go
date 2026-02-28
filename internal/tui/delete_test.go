package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtsunne/epm-go/internal/client"
	"github.com/jtsunne/epm-go/internal/model"
)

// tuiMockClient is a minimal ESClient implementation for tui-package tests.
type tuiMockClient struct {
	deleteIndexFn func(ctx context.Context, names []string) error
}

func (m *tuiMockClient) GetClusterHealth(ctx context.Context) (*client.ClusterHealth, error) {
	return &client.ClusterHealth{}, nil
}
func (m *tuiMockClient) GetNodes(ctx context.Context) ([]client.NodeInfo, error) {
	return nil, nil
}
func (m *tuiMockClient) GetNodeStats(ctx context.Context) (*client.NodeStatsResponse, error) {
	return &client.NodeStatsResponse{Nodes: map[string]client.NodePerformanceStats{}}, nil
}
func (m *tuiMockClient) GetIndices(ctx context.Context) ([]client.IndexInfo, error) {
	return nil, nil
}
func (m *tuiMockClient) GetIndexStats(ctx context.Context) (*client.IndexStatsResponse, error) {
	return &client.IndexStatsResponse{Indices: map[string]client.IndexStatEntry{}}, nil
}
func (m *tuiMockClient) DeleteIndex(ctx context.Context, names []string) error {
	if m.deleteIndexFn != nil {
		return m.deleteIndexFn(ctx, names)
	}
	return nil
}
func (m *tuiMockClient) Ping(ctx context.Context) error { return nil }
func (m *tuiMockClient) BaseURL() string                { return "http://mock:9200" }

// TestDeleteCmd_Success verifies that deleteCmd returns a DeleteResultMsg with
// nil Err and the correct Names on a successful DeleteIndex call.
func TestDeleteCmd_Success(t *testing.T) {
	mc := &tuiMockClient{
		deleteIndexFn: func(_ context.Context, names []string) error {
			return nil
		},
	}
	names := []string{"logs-2024", "metrics-daily"}
	cmd := deleteCmd(mc, names)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(DeleteResultMsg)
	require.True(t, ok, "expected DeleteResultMsg, got %T", msg)
	assert.Equal(t, names, result.Names)
	assert.NoError(t, result.Err)
}

// TestDeleteCmd_Error verifies that deleteCmd propagates the error from
// DeleteIndex in the returned DeleteResultMsg.
func TestDeleteCmd_Error(t *testing.T) {
	want := errors.New("index not found")
	mc := &tuiMockClient{
		deleteIndexFn: func(_ context.Context, names []string) error {
			return want
		},
	}
	names := []string{"missing-index"}
	cmd := deleteCmd(mc, names)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(DeleteResultMsg)
	require.True(t, ok)
	assert.Equal(t, want, result.Err)
	assert.Equal(t, names, result.Names)
}

// TestRenderDeleteConfirm_ContainsIndexNames verifies that the confirmation
// view renders the names of all pending deletion indices.
func TestRenderDeleteConfirm_ContainsIndexNames(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120
	app.height = 40
	app.pendingDeleteNames = []string{"logs-2024", "metrics-daily"}

	out := renderDeleteConfirm(app)
	stripped := stripANSI(out)
	assert.True(t, strings.Contains(stripped, "logs-2024"), "confirm view must contain first index name")
	assert.True(t, strings.Contains(stripped, "metrics-daily"), "confirm view must contain second index name")
	assert.True(t, strings.Contains(stripped, "WARNING"), "confirm view must contain warning text")
}

// TestRenderDeleteConfirm_SingleIndex verifies cursor-row single-index mode.
func TestRenderDeleteConfirm_SingleIndex(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 100
	app.height = 30
	app.pendingDeleteNames = []string{"my-only-index"}

	out := renderDeleteConfirm(app)
	stripped := stripANSI(out)
	assert.True(t, strings.Contains(stripped, "my-only-index"), "single-index confirm view must show the index name")
	assert.True(t, strings.Contains(stripped, "1 index(es)"), "single-index confirm must say 1 index")
}

// TestApp_DeleteKey_EntersConfirmMode verifies that pressing d when the index
// table is active and has a row under the cursor sets deleteConfirmMode.
func TestApp_DeleteKey_EntersConfirmMode(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.activeTable = 0
	app.indexTable.focused = true

	// Add a row via SetData so the cursor has something to target.
	app.indexTable.SetData([]model.IndexRow{
		{Name: "target-index", IndexingRate: 100.0},
	})

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := newModel.(*App)

	assert.True(t, updated.deleteConfirmMode, "d key should enter deleteConfirmMode")
	require.Len(t, updated.pendingDeleteNames, 1)
	assert.Equal(t, "target-index", updated.pendingDeleteNames[0])
}

// TestApp_DeleteKey_UsesSelectionOverCursor verifies that when rows are
// selected, d uses the selection list rather than the cursor row.
func TestApp_DeleteKey_UsesSelectionOverCursor(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.activeTable = 0
	app.indexTable.focused = true
	app.indexTable.SetData([]model.IndexRow{
		{Name: "alpha", IndexingRate: 300.0},
		{Name: "beta", IndexingRate: 200.0},
		{Name: "gamma", IndexingRate: 100.0},
	})
	// Select beta and gamma explicitly.
	app.indexTable.toggleSelect("beta")
	app.indexTable.toggleSelect("gamma")

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := newModel.(*App)

	assert.True(t, updated.deleteConfirmMode)
	// selectedNames() returns sorted slice: beta, gamma
	assert.Equal(t, []string{"beta", "gamma"}, updated.pendingDeleteNames)
}

// TestApp_DeleteKey_NoopOnNodeTable verifies that d is a no-op when the node
// table is active.
func TestApp_DeleteKey_NoopOnNodeTable(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.activeTable = 1
	app.nodeTable.focused = true

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := newModel.(*App)

	assert.False(t, updated.deleteConfirmMode, "d on node table must not enter deleteConfirmMode")
}

// TestApp_DeleteKey_NoopWhenNoRows verifies that d does nothing when the index
// table is empty (no rows to delete).
func TestApp_DeleteKey_NoopWhenNoRows(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.activeTable = 0
	app.indexTable.focused = true
	// Do not call SetData — displayRows is empty.

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated := newModel.(*App)

	assert.False(t, updated.deleteConfirmMode, "d on empty table must not enter deleteConfirmMode")
}

// TestApp_DeleteConfirmMode_YExecutesDelete verifies that pressing y in
// confirmMode clears the mode, clears the selection, and returns a tea.Cmd.
func TestApp_DeleteConfirmMode_YExecutesDelete(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.deleteConfirmMode = true
	app.pendingDeleteNames = []string{"logs-2024"}
	app.indexTable.toggleSelect("logs-2024")

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	updated := newModel.(*App)

	assert.False(t, updated.deleteConfirmMode, "confirm mode should exit after y")
	assert.Nil(t, updated.pendingDeleteNames, "pendingDeleteNames should be cleared after y")
	assert.Empty(t, updated.indexTable.selected, "selection should be cleared after y")
	require.NotNil(t, cmd, "y should return a delete command")
}

// TestApp_DeleteConfirmMode_NExitsWithoutDelete verifies that pressing n exits
// the confirm mode without issuing any command.
func TestApp_DeleteConfirmMode_NExitsWithoutDelete(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.deleteConfirmMode = true
	app.pendingDeleteNames = []string{"logs-2024"}

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updated := newModel.(*App)

	assert.False(t, updated.deleteConfirmMode, "confirm mode should exit after n")
	assert.Nil(t, updated.pendingDeleteNames, "pendingDeleteNames should be cleared after n")
	assert.Nil(t, cmd, "n must not issue any command")
}

// TestApp_DeleteConfirmMode_EscExitsWithoutDelete verifies that esc exits the
// confirm mode without issuing any command.
func TestApp_DeleteConfirmMode_EscExitsWithoutDelete(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.deleteConfirmMode = true
	app.pendingDeleteNames = []string{"logs-2024"}

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updated := newModel.(*App)

	assert.False(t, updated.deleteConfirmMode, "confirm mode should exit on esc")
	assert.Nil(t, updated.pendingDeleteNames)
	assert.Nil(t, cmd)
}

// TestApp_DeleteConfirmMode_OtherKeysBlocked verifies that unrelated keys are
// blocked while deleteConfirmMode is active.
func TestApp_DeleteConfirmMode_OtherKeysBlocked(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.deleteConfirmMode = true
	app.pendingDeleteNames = []string{"logs-2024"}

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	updated := newModel.(*App)

	assert.True(t, updated.deleteConfirmMode, "confirm mode must persist for unrelated keys")
	assert.Nil(t, cmd)
}

// TestApp_DeleteResultMsg_Success verifies that a successful DeleteResultMsg
// sets deleteStatus and triggers a fetch command.
func TestApp_DeleteResultMsg_Success(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.fetching = false

	names := []string{"logs-2024"}
	newModel, cmd := app.Update(DeleteResultMsg{Names: names, Err: nil})
	updated := newModel.(*App)

	assert.Equal(t, "Deleted 1 index(es)", updated.deleteStatus)
	assert.True(t, updated.fetching, "a successful delete should trigger an immediate fetch")
	require.NotNil(t, cmd)
}

// TestApp_DeleteResultMsg_Error verifies that a failed DeleteResultMsg sets
// the error deleteStatus and does not trigger a fetch.
func TestApp_DeleteResultMsg_Error(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.fetching = false

	err := errors.New("permission denied")
	newModel, cmd := app.Update(DeleteResultMsg{Names: []string{"logs"}, Err: err})
	updated := newModel.(*App)

	assert.True(t, strings.HasPrefix(updated.deleteStatus, "Delete failed"), "error status must start with 'Delete failed'")
	assert.False(t, updated.fetching, "a failed delete must not trigger a fetch")
	assert.Nil(t, cmd)
}

// TestApp_SnapshotMsg_ClearsDeleteStatus verifies that a SnapshotMsg clears
// any pending deleteStatus so the footer returns to normal display.
func TestApp_SnapshotMsg_ClearsDeleteStatus(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.deleteStatus = "Deleted 2 index(es)"

	snap := makeFixtureSnapshot()
	newModel, _ := app.Update(makeFixtureMsg(snap))
	updated := newModel.(*App)

	assert.Empty(t, updated.deleteStatus, "SnapshotMsg must clear deleteStatus")
}

// TestRenderFooter_ShowsDeleteStatusGreen verifies the footer shows a green
// success message when deleteStatus is set and does not start with "Delete failed".
func TestRenderFooter_ShowsDeleteStatusGreen(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 80
	app.deleteStatus = "Deleted 2 index(es)"

	out := stripANSI(renderFooter(app))
	assert.Contains(t, out, "Deleted 2 index(es)")
}

// TestRenderFooter_ShowsDeleteStatusRed verifies the footer shows an error
// message when deleteStatusErr is true.
func TestRenderFooter_ShowsDeleteStatusRed(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 80
	app.deleteStatus = "Delete failed: permission denied"
	app.deleteStatusErr = true

	out := stripANSI(renderFooter(app))
	assert.Contains(t, out, "Delete failed: permission denied")
}

// TestApp_FetchErrorMsg_ClearsDeleteStatus verifies that a FetchErrorMsg
// clears any pending deleteStatus so stale messages don't linger in the footer.
func TestApp_FetchErrorMsg_ClearsDeleteStatus(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.deleteStatus = "Delete failed: some error"
	app.deleteStatusErr = true

	newModel, _ := app.Update(FetchErrorMsg{Err: errors.New("connection refused")})
	updated := newModel.(*App)

	assert.Empty(t, updated.deleteStatus, "FetchErrorMsg must clear deleteStatus")
	assert.False(t, updated.deleteStatusErr, "FetchErrorMsg must clear deleteStatusErr")
}

// TestApp_DeleteConfirmMode_OtherKeysBlocked_PreservesPendingNames verifies
// that unrelated keys do not clear pendingDeleteNames.
func TestApp_DeleteConfirmMode_OtherKeysBlocked_PreservesPendingNames(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.deleteConfirmMode = true
	app.pendingDeleteNames = []string{"logs-2024"}

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	updated := newModel.(*App)

	assert.True(t, updated.deleteConfirmMode, "confirm mode must persist for unrelated keys")
	assert.Equal(t, []string{"logs-2024"}, updated.pendingDeleteNames, "pendingDeleteNames must be preserved")
	assert.Nil(t, cmd)
}

// TestApp_DeleteResultMsg_SetsDeleteStatusErr verifies that deleteStatusErr is
// true on error and false on success.
func TestApp_DeleteResultMsg_SetsDeleteStatusErr(t *testing.T) {
	mc := &tuiMockClient{}

	t.Run("error sets deleteStatusErr true", func(t *testing.T) {
		app := NewApp(mc, 10*time.Second)
		newModel, _ := app.Update(DeleteResultMsg{Names: []string{"logs"}, Err: errors.New("denied")})
		updated := newModel.(*App)
		assert.True(t, updated.deleteStatusErr)
	})

	t.Run("success sets deleteStatusErr false", func(t *testing.T) {
		app := NewApp(mc, 10*time.Second)
		newModel, _ := app.Update(DeleteResultMsg{Names: []string{"logs"}, Err: nil})
		updated := newModel.(*App)
		assert.False(t, updated.deleteStatusErr)
	})
}

// TestApp_DeleteResultMsg_Success_WhileFetching verifies that when a successful
// DeleteResultMsg arrives while a stale fetch is in-flight, pendingRefresh is
// set (rather than issuing an immediate fetch) and no command is returned.
func TestApp_DeleteResultMsg_Success_WhileFetching(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.fetching = true // simulate a stale fetch already in-flight

	newModel, cmd := app.Update(DeleteResultMsg{Names: []string{"logs-2024"}, Err: nil})
	updated := newModel.(*App)

	assert.True(t, updated.pendingRefresh, "pendingRefresh must be set when fetch is already in-flight")
	assert.Nil(t, cmd, "no immediate fetch cmd should be issued while a fetch is in-flight")
	assert.Equal(t, "Deleted 1 index(es)", updated.deleteStatus)
}

// TestApp_SnapshotMsg_PendingRefresh_TriggersNewFetch verifies that when
// pendingRefresh is true, a SnapshotMsg: (a) does NOT clear deleteStatus,
// (b) clears pendingRefresh, and (c) issues an immediate fetchCmd.
func TestApp_SnapshotMsg_PendingRefresh_TriggersNewFetch(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.pendingRefresh = true
	app.deleteStatus = "Deleted 1 index(es)"

	snap := makeFixtureSnapshot()
	newModel, cmd := app.Update(makeFixtureMsg(snap))
	updated := newModel.(*App)

	assert.False(t, updated.pendingRefresh, "pendingRefresh must be cleared after SnapshotMsg")
	assert.Equal(t, "Deleted 1 index(es)", updated.deleteStatus, "deleteStatus must not be cleared by stale SnapshotMsg")
	require.NotNil(t, cmd, "an immediate fetch cmd must be issued after the stale snapshot lands")
}

// TestRenderDeleteConfirm_UltraSmallHeight_NoOverflow verifies that at
// height=4 (availH=1, fLen=2) the content body never produces more lines
// than availH, preventing the full view from exceeding the terminal height.
func TestRenderDeleteConfirm_UltraSmallHeight_NoOverflow(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 80
	app.height = 4
	app.pendingDeleteNames = []string{"logs-2024", "metrics-daily"}

	out := renderDeleteConfirm(app)
	// renderDeleteConfirm returns titleBar + "\n" + content.
	// Count the lines it produces; must not exceed titleH (1) + availH (1) = 2.
	lines := strings.Split(out, "\n")
	headerH := renderedHeight(renderHeader(app))
	footerH := renderedHeight(renderFooter(app))
	availH := app.height - headerH - footerH
	if availH < 1 {
		availH = 1
	}
	// renderDeleteConfirm output = titleH + actual_availH lines.
	// test's availH already equals titleH + actual_availH (since it omits titleH
	// from the subtraction), so the output must fit within availH lines exactly.
	maxLines := availH
	assert.LessOrEqual(t, len(lines), maxLines,
		"renderDeleteConfirm must not produce more than %d lines at height=4 (availH=%d)", maxLines, availH)
}

// TestApp_FetchErrorMsg_ClearsPendingRefresh verifies that a FetchErrorMsg
// clears pendingRefresh so the stale-fetch refresh loop does not get stuck.
func TestApp_FetchErrorMsg_ClearsPendingRefresh(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.pendingRefresh = true

	newModel, _ := app.Update(FetchErrorMsg{Err: errors.New("connection refused")})
	updated := newModel.(*App)

	assert.False(t, updated.pendingRefresh, "FetchErrorMsg must clear pendingRefresh")
}

// TestRenderDeleteConfirm_SmallHeight_FooterAlwaysVisible verifies that the
// "Press y to confirm" prompt is always present even at very small terminal
// heights (the fix for the availH <= 7 footer-trim bug and the ultra-small
// height overflow where fLen > availH).
func TestRenderDeleteConfirm_SmallHeight_FooterAlwaysVisible(t *testing.T) {
	for _, h := range []int{4, 5, 6, 7, 8, 9, 10} {
		app := NewApp(nil, 10*time.Second)
		app.width = 80
		app.height = h
		app.pendingDeleteNames = []string{"logs-2024", "metrics-daily", "traces-old"}

		out := renderDeleteConfirm(app)
		stripped := stripANSI(out)
		assert.True(t, strings.Contains(stripped, "Press y to confirm"),
			"footer prompt must be visible at height=%d", h)
	}
}
