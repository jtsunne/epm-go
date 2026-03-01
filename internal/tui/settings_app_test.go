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

// TestSettingsLoadCmd_Success verifies that settingsLoadCmd returns a
// SettingsLoadedMsg with the values from GetIndexSettings on success.
func TestSettingsLoadCmd_Success(t *testing.T) {
	want := &client.IndexSettingsValues{NumberOfReplicas: "2", RefreshInterval: "30s"}
	mc := &tuiMockClient{
		getIndexSettingsFn: func(_ context.Context, _ string) (*client.IndexSettingsValues, error) {
			return want, nil
		},
	}
	cmd := settingsLoadCmd(mc, "my-index", 0)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(SettingsLoadedMsg)
	require.True(t, ok, "expected SettingsLoadedMsg, got %T", msg)
	assert.Equal(t, want, result.Values)
	assert.NoError(t, result.Err)
}

// TestSettingsLoadCmd_Error verifies that settingsLoadCmd propagates errors.
func TestSettingsLoadCmd_Error(t *testing.T) {
	want := errors.New("index not found")
	mc := &tuiMockClient{
		getIndexSettingsFn: func(_ context.Context, _ string) (*client.IndexSettingsValues, error) {
			return nil, want
		},
	}
	cmd := settingsLoadCmd(mc, "missing-index", 0)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(SettingsLoadedMsg)
	require.True(t, ok)
	assert.Equal(t, want, result.Err)
	assert.Nil(t, result.Values)
}

// TestSettingsUpdateCmd_Success verifies that settingsUpdateCmd returns a
// SettingsResultMsg with nil Err on success.
func TestSettingsUpdateCmd_Success(t *testing.T) {
	mc := &tuiMockClient{}
	names := []string{"logs-2024", "metrics-daily"}
	settings := map[string]any{"index.number_of_replicas": "2"}

	cmd := settingsUpdateCmd(mc, names, settings, 0)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(SettingsResultMsg)
	require.True(t, ok, "expected SettingsResultMsg, got %T", msg)
	assert.Equal(t, names, result.Names)
	assert.NoError(t, result.Err)
}

// TestSettingsUpdateCmd_Error verifies that settingsUpdateCmd propagates errors.
func TestSettingsUpdateCmd_Error(t *testing.T) {
	want := errors.New("settings rejected")
	mc := &tuiMockClient{
		updateIndexSettingsFn: func(_ context.Context, _ []string, _ map[string]any) error {
			return want
		},
	}
	cmd := settingsUpdateCmd(mc, []string{"logs"}, map[string]any{"index.number_of_replicas": "3"}, 0)
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(SettingsResultMsg)
	require.True(t, ok)
	assert.Equal(t, want, result.Err)
}

// TestApp_EditKey_EntersSettingsMode verifies that pressing e when the index
// table is active and has a row under the cursor enters settingsMode and
// returns a settingsLoadCmd command.
func TestApp_EditKey_EntersSettingsMode(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.activeTable = 0
	app.indexTable.focused = true
	app.indexTable.SetData([]model.IndexRow{
		{Name: "target-index", IndexingRate: 100.0},
	})

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	updated := newModel.(*App)

	assert.True(t, updated.settingsMode, "e key should enter settingsMode")
	require.NotNil(t, cmd, "e key should return a settings load command")
	// The command should be a settingsLoadCmd returning SettingsLoadedMsg.
	msg := cmd()
	_, ok := msg.(SettingsLoadedMsg)
	assert.True(t, ok, "command should produce SettingsLoadedMsg, got %T", msg)
}

// TestApp_EditKey_UsesSelectionOverCursor verifies that when rows are
// selected, e uses the selection list (form names = selected, load uses first).
func TestApp_EditKey_UsesSelectionOverCursor(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.activeTable = 0
	app.indexTable.focused = true
	app.indexTable.SetData([]model.IndexRow{
		{Name: "alpha"},
		{Name: "beta"},
	})
	app.indexTable.toggleSelect("alpha")
	app.indexTable.toggleSelect("beta")

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	updated := newModel.(*App)

	assert.True(t, updated.settingsMode)
	assert.Equal(t, []string{"alpha", "beta"}, updated.settingsForm.names)
}

// TestApp_EditKey_NoopOnNodeTable verifies that e is a no-op when the node
// table is active.
func TestApp_EditKey_NoopOnNodeTable(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.activeTable = 1
	app.nodeTable.focused = true

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	updated := newModel.(*App)

	assert.False(t, updated.settingsMode, "e on node table must not enter settingsMode")
	assert.Nil(t, cmd)
}

// TestApp_EditKey_NoopWhenEmpty verifies that e does nothing on an empty index table.
func TestApp_EditKey_NoopWhenEmpty(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.activeTable = 0
	app.indexTable.focused = true

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	updated := newModel.(*App)

	assert.False(t, updated.settingsMode, "e on empty table must not enter settingsMode")
	assert.Nil(t, cmd)
}

// TestApp_SettingsLoadedMsg_AppliesValues verifies that SettingsLoadedMsg
// calls applySettings and clears the loading flag.
func TestApp_SettingsLoadedMsg_AppliesValues(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"my-index"}, nil, nil)

	vals := &client.IndexSettingsValues{
		NumberOfReplicas: "3",
		RefreshInterval:  "60s",
	}
	newModel, _ := app.Update(SettingsLoadedMsg{Values: vals, Err: nil})
	updated := newModel.(*App)

	assert.False(t, updated.settingsForm.loading, "loading should be false after SettingsLoadedMsg")
	f := findField(updated.settingsForm, "index.number_of_replicas")
	require.NotNil(t, f)
	assert.Equal(t, "3", f.currentVal)
}

// TestApp_SettingsLoadedMsg_Error verifies that a SettingsLoadedMsg with an
// error sets loadErr and clears loading.
func TestApp_SettingsLoadedMsg_Error(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"my-index"}, nil, nil)

	err := errors.New("forbidden")
	newModel, _ := app.Update(SettingsLoadedMsg{Values: nil, Err: err})
	updated := newModel.(*App)

	assert.False(t, updated.settingsForm.loading)
	assert.Contains(t, updated.settingsForm.loadErr, "forbidden")
}

// TestApp_SettingsResultMsg_Success verifies that a successful SettingsResultMsg
// exits settingsMode, sets settingsStatus, and issues a fetch.
func TestApp_SettingsResultMsg_Success(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.settingsMode = true
	app.fetching = false

	names := []string{"logs-2024"}
	newModel, cmd := app.Update(SettingsResultMsg{Names: names, Err: nil})
	updated := newModel.(*App)

	assert.False(t, updated.settingsMode, "settingsMode should exit after SettingsResultMsg")
	assert.Contains(t, updated.settingsStatus, "Settings updated")
	assert.False(t, updated.settingsStatusErr)
	assert.True(t, updated.fetching, "a successful settings update should trigger an immediate fetch")
	require.NotNil(t, cmd)
}

// TestApp_SettingsResultMsg_Error verifies that a failed SettingsResultMsg sets
// the error settingsStatus and does not trigger a fetch.
func TestApp_SettingsResultMsg_Error(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.settingsMode = true
	app.fetching = false

	err := errors.New("permission denied")
	newModel, cmd := app.Update(SettingsResultMsg{Names: []string{"logs"}, Err: err})
	updated := newModel.(*App)

	assert.False(t, updated.settingsMode, "settingsMode should exit even on error")
	assert.Contains(t, updated.settingsStatus, "Settings update failed")
	assert.True(t, updated.settingsStatusErr)
	assert.False(t, updated.fetching)
	assert.Nil(t, cmd)
}

// TestApp_SettingsResultMsg_Success_WhileFetching verifies that when a
// successful SettingsResultMsg arrives while a fetch is already in-flight,
// settingsPendingRefresh is set (no duplicate fetch issued) and settingsStatus
// is preserved until the pending refresh clears it.
func TestApp_SettingsResultMsg_Success_WhileFetching(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.settingsMode = true
	app.fetching = true // stale fetch already in-flight

	names := []string{"logs-2024"}
	newModel, cmd := app.Update(SettingsResultMsg{Names: names, Err: nil})
	updated := newModel.(*App)

	assert.False(t, updated.settingsMode)
	assert.Contains(t, updated.settingsStatus, "Settings updated")
	assert.True(t, updated.fetching, "existing in-flight fetch must remain active")
	assert.True(t, updated.settingsPendingRefresh, "settingsPendingRefresh must be set")
	assert.Nil(t, cmd, "no new fetch cmd should be returned while already fetching")
}

// TestApp_SnapshotMsg_SettingsPendingRefresh verifies that when settingsPendingRefresh
// is true, the stale SnapshotMsg does NOT clear settingsStatus and instead triggers
// a new fetch.
func TestApp_SnapshotMsg_SettingsPendingRefresh(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.settingsStatus = "Settings updated for 1 index(es)"
	app.settingsPendingRefresh = true

	snap := makeFixtureSnapshot()
	newModel, cmd := app.Update(makeFixtureMsg(snap))
	updated := newModel.(*App)

	assert.NotEmpty(t, updated.settingsStatus, "settingsStatus must NOT be cleared by the stale SnapshotMsg")
	assert.False(t, updated.settingsPendingRefresh, "settingsPendingRefresh must be cleared after triggering refresh")
	assert.True(t, updated.fetching, "a new fetch must be triggered after the stale snapshot lands")
	require.NotNil(t, cmd)
}

// TestApp_SettingsMode_EscExitsMode verifies that esc in settingsMode exits
// the form without issuing any command.
func TestApp_SettingsMode_EscExitsMode(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"idx"}, nil, nil)
	app.settingsForm.loading = false

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updated := newModel.(*App)

	assert.False(t, updated.settingsMode, "esc should exit settingsMode")
	assert.Nil(t, cmd)
}

// TestApp_SettingsMode_OtherKeysRouted verifies that non-exit keys in
// settingsMode are routed to the form and don't leave settingsMode.
func TestApp_SettingsMode_OtherKeysRouted(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"idx"}, nil, nil)
	app.settingsForm.loading = false // allow navigation

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated := newModel.(*App)

	// settingsMode must still be active; focusedField must have advanced.
	assert.True(t, updated.settingsMode)
	assert.Equal(t, 1, updated.settingsForm.focusedField)
}

// TestApp_SnapshotMsg_ClearsSettingsStatus verifies that a SnapshotMsg clears
// any pending settingsStatus so the footer returns to normal display.
func TestApp_SnapshotMsg_ClearsSettingsStatus(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.settingsStatus = "Settings updated for 1 index(es)"
	app.settingsStatusErr = false

	snap := makeFixtureSnapshot()
	newModel, _ := app.Update(makeFixtureMsg(snap))
	updated := newModel.(*App)

	assert.Empty(t, updated.settingsStatus, "SnapshotMsg must clear settingsStatus")
	assert.False(t, updated.settingsStatusErr)
}

// TestApp_FetchErrorMsg_ClearsSettingsStatus verifies that a FetchErrorMsg
// clears any pending settingsStatus.
func TestApp_FetchErrorMsg_ClearsSettingsStatus(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.settingsStatus = "Settings update failed: denied"
	app.settingsStatusErr = true

	newModel, _ := app.Update(FetchErrorMsg{Err: errors.New("timeout")})
	updated := newModel.(*App)

	assert.Empty(t, updated.settingsStatus, "FetchErrorMsg must clear settingsStatus")
	assert.False(t, updated.settingsStatusErr)
}

// TestRenderFooter_ShowsSettingsStatusGreen verifies the footer shows a green
// success message when settingsStatus is set without error.
func TestRenderFooter_ShowsSettingsStatusGreen(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 80
	app.settingsStatus = "Settings updated for 1 index(es)"

	out := stripANSI(renderFooter(app))
	assert.Contains(t, out, "Settings updated for 1 index(es)")
}

// TestRenderFooter_ShowsSettingsStatusRed verifies the footer shows an error
// message when settingsStatusErr is true.
func TestRenderFooter_ShowsSettingsStatusRed(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 80
	app.settingsStatus = "Settings update failed: denied"
	app.settingsStatusErr = true

	out := stripANSI(renderFooter(app))
	assert.Contains(t, out, "Settings update failed: denied")
}

// TestRenderSettingsForm_Loading verifies that the loading state is shown
// when the form has loading=true.
func TestRenderSettingsForm_Loading(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120
	app.height = 40
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"my-index"}, nil, nil)
	// loading is true by default from buildSettingsForm

	out := stripANSI(renderSettingsForm(app))
	assert.Contains(t, out, "Loading current settings...")
}

// TestRenderSettingsForm_ShowsFieldLabels verifies that the form displays
// the expected field labels after settings are loaded.
func TestRenderSettingsForm_ShowsFieldLabels(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120
	app.height = 60
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"my-index"}, nil, nil)
	app.settingsForm.loading = false

	out := stripANSI(renderSettingsForm(app))
	assert.Contains(t, out, "Replicas")
	assert.Contains(t, out, "Refresh Interval")
}

// TestRenderSettingsForm_TitleIncludesIndexName verifies that the title bar
// shows the index name being edited.
func TestRenderSettingsForm_TitleIncludesIndexName(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120
	app.height = 40
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"my-index"}, nil, nil)

	out := stripANSI(renderSettingsForm(app))
	assert.Contains(t, out, "my-index")
}

// TestRenderSettingsForm_MultiIndexTitle verifies that the title bar says
// "N indices" when more than one index is being edited.
func TestRenderSettingsForm_MultiIndexTitle(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120
	app.height = 40
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"idx-a", "idx-b"}, nil, nil)

	out := stripANSI(renderSettingsForm(app))
	assert.True(t, strings.Contains(out, "2 indices"), "title must say '2 indices' for multi-index edit, got: %s", out)
}

// TestRenderSettingsForm_ErrorState verifies that the load error message is shown.
func TestRenderSettingsForm_ErrorState(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 120
	app.height = 40
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"my-index"}, nil, nil)
	app.settingsForm.loading = false
	app.settingsForm.loadErr = "access denied"

	out := stripANSI(renderSettingsForm(app))
	assert.Contains(t, out, "Error: access denied")
}

// TestApp_SettingsMode_CtrlS_NoChanges verifies that ctrl+s with no changed
// fields exits settingsMode without issuing a network command.
func TestApp_SettingsMode_CtrlS_NoChanges(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"idx"}, nil, nil)
	// Apply same values so changedSettings() is empty.
	app.settingsForm.applySettings(&client.IndexSettingsValues{NumberOfReplicas: "1"})

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	updated := newModel.(*App)

	assert.False(t, updated.settingsMode, "ctrl+s with no changes should exit settingsMode")
	assert.Nil(t, cmd, "no command should be issued when nothing changed")
}

// TestApp_SettingsMode_CtrlS_WithChanges verifies that ctrl+s with changes
// issues a settingsUpdateCmd and remains in settingsMode until the result arrives.
func TestApp_SettingsMode_CtrlS_WithChanges(t *testing.T) {
	mc := &tuiMockClient{}
	app := NewApp(mc, 10*time.Second)
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"idx"}, nil, nil)
	app.settingsForm.applySettings(&client.IndexSettingsValues{NumberOfReplicas: "1"})
	// Change the replicas field.
	setFieldInputValue(&app.settingsForm, "index.number_of_replicas", "3")

	newModel, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	updated := newModel.(*App)

	// settingsMode exits immediately on submit to prevent double-submit.
	assert.False(t, updated.settingsMode, "settingsMode must exit immediately on ctrl+s with changes")
	require.NotNil(t, cmd, "ctrl+s with changes must return a settingsUpdateCmd")
	// Execute it and verify it produces a SettingsResultMsg.
	msg := cmd()
	_, ok := msg.(SettingsResultMsg)
	assert.True(t, ok, "command should produce SettingsResultMsg, got %T", msg)
}

// TestApp_StaleSettingsLoadedMsg_Ignored verifies that a SettingsLoadedMsg with
// a stale nonce does not overwrite the active form's state.
func TestApp_StaleSettingsLoadedMsg_Ignored(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	// Simulate opening a second session: settingsNonce = 2.
	app.settingsNonce = 2
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"new-index"}, nil, nil)

	// Stale response from the first session carries nonce=1.
	staleVals := &client.IndexSettingsValues{NumberOfReplicas: "99"}
	newModel, _ := app.Update(SettingsLoadedMsg{Values: staleVals, Err: nil, Nonce: 1})
	updated := newModel.(*App)

	// Form must still be in loading state — the stale message must not apply.
	assert.True(t, updated.settingsForm.loading, "stale SettingsLoadedMsg must not change loading state")
	f := findField(updated.settingsForm, "index.number_of_replicas")
	require.NotNil(t, f)
	assert.Empty(t, f.currentVal, "stale SettingsLoadedMsg must not populate currentVal")
}

// TestApp_StaleSettingsResultMsg_Ignored verifies that a SettingsResultMsg with
// a stale nonce does not close an active settings session.
func TestApp_StaleSettingsResultMsg_Ignored(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	// Session nonce=2 is active.
	app.settingsNonce = 2
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"new-index"}, nil, nil)

	// Stale result from session nonce=1.
	newModel, cmd := app.Update(SettingsResultMsg{Names: []string{"old-index"}, Err: nil, Nonce: 1})
	updated := newModel.(*App)

	assert.True(t, updated.settingsMode, "stale SettingsResultMsg must not close the active settings form")
	assert.Nil(t, cmd, "stale SettingsResultMsg must not trigger a fetch")
	assert.Empty(t, updated.settingsStatus, "stale SettingsResultMsg must not set settingsStatus")
}

// TestRenderSettingsForm_SmallHeight verifies that rendering does not panic or
// produce more lines than the available terminal height when height is very small.
func TestRenderSettingsForm_SmallHeight_NoOverflow(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 80
	app.height = 12 // very small — most fields will be truncated
	app.settingsMode = true
	app.settingsForm = buildSettingsForm([]string{"my-index"}, nil, nil)
	app.settingsForm.loading = false

	out := renderSettingsForm(app)
	lines := strings.Split(out, "\n")
	// The rendered output must not exceed the terminal height.
	assert.LessOrEqual(t, len(lines), app.height,
		"renderSettingsForm must not overflow terminal height, got %d lines for height %d", len(lines), app.height)
}
