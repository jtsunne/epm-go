package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtsunne/epm-go/internal/client"
)

// TestBuildSettingsForm_FieldCount verifies that buildSettingsForm creates
// exactly 11 fields — one for each exposed ES setting.
func TestBuildSettingsForm_FieldCount(t *testing.T) {
	m := buildSettingsForm([]string{"my-index"}, nil, nil)
	assert.Len(t, m.fields, 11, "settings form must have 11 fields")
}

// TestBuildSettingsForm_InitialState verifies loading=true, focusedField=0,
// and that the first field has its input focused.
func TestBuildSettingsForm_InitialState(t *testing.T) {
	m := buildSettingsForm([]string{"my-index"}, nil, nil)
	assert.True(t, m.loading, "form must start in loading state")
	assert.Equal(t, 0, m.focusedField)
	assert.Equal(t, []string{"my-index"}, m.names)
}

// TestBuildSettingsForm_RoutingSuggestions verifies that nodeNames are set as
// suggestions for the three routing include/exclude/require _name fields.
func TestBuildSettingsForm_RoutingSuggestions(t *testing.T) {
	nodeNames := []string{"node-1", "node-2"}
	nodeIPs := []string{"10.0.0.1", "10.0.0.2"}
	m := buildSettingsForm([]string{"idx"}, nodeNames, nodeIPs)

	nameKeys := []string{
		"index.routing.allocation.include._name",
		"index.routing.allocation.exclude._name",
		"index.routing.allocation.require._name",
	}
	ipKeys := []string{
		"index.routing.allocation.include._ip",
		"index.routing.allocation.exclude._ip",
		"index.routing.allocation.require._ip",
	}

	for _, k := range nameKeys {
		f := findField(m, k)
		require.NotNil(t, f, "field %s must exist", k)
		assert.Equal(t, nodeNames, f.suggestions, "field %s must have nodeNames as suggestions", k)
	}
	for _, k := range ipKeys {
		f := findField(m, k)
		require.NotNil(t, f, "field %s must exist", k)
		assert.Equal(t, nodeIPs, f.suggestions, "field %s must have nodeIPs as suggestions", k)
	}
}

// TestBuildSettingsForm_AllESKeys verifies that all 11 expected ES keys are present.
func TestBuildSettingsForm_AllESKeys(t *testing.T) {
	m := buildSettingsForm(nil, nil, nil)
	expectedKeys := []string{
		"index.number_of_replicas",
		"index.refresh_interval",
		"index.routing.allocation.include._name",
		"index.routing.allocation.exclude._name",
		"index.routing.allocation.require._name",
		"index.routing.allocation.include._ip",
		"index.routing.allocation.exclude._ip",
		"index.routing.allocation.require._ip",
		"index.routing.allocation.total_shards_per_node",
		"index.mapping.total_fields.limit",
		"index.blocks.read_only_allow_delete",
	}
	for _, key := range expectedKeys {
		f := findField(m, key)
		assert.NotNil(t, f, "field with ESKey %q must exist", key)
	}
}

// TestApplySettings_PopulatesFields verifies that applySettings sets currentVal
// and the input value for each mapped field.
func TestApplySettings_PopulatesFields(t *testing.T) {
	m := buildSettingsForm([]string{"my-index"}, nil, nil)
	vals := &client.IndexSettingsValues{
		NumberOfReplicas: "2",
		RefreshInterval:  "30s",
		Routing: client.IndexRoutingSettings{
			Allocation: client.IndexAllocationSettings{
				Include: client.IndexAllocationFilter{Name: "node-1"},
				Exclude: client.IndexAllocationFilter{IP: "10.0.0.5"},
			},
		},
	}

	m.applySettings(vals)

	assert.False(t, m.loading, "loading must be false after applySettings")

	f := findField(m, "index.number_of_replicas")
	require.NotNil(t, f)
	assert.Equal(t, "2", f.currentVal)
	assert.Equal(t, "2", f.input.Value())

	f = findField(m, "index.refresh_interval")
	require.NotNil(t, f)
	assert.Equal(t, "30s", f.currentVal)
	assert.Equal(t, "30s", f.input.Value())

	f = findField(m, "index.routing.allocation.include._name")
	require.NotNil(t, f)
	assert.Equal(t, "node-1", f.currentVal)

	f = findField(m, "index.routing.allocation.exclude._ip")
	require.NotNil(t, f)
	assert.Equal(t, "10.0.0.5", f.currentVal)
}

// TestApplySettings_NilValues verifies that applySettings with nil just
// clears the loading state without panicking.
func TestApplySettings_NilValues(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	m.applySettings(nil)
	assert.False(t, m.loading)
}

// TestChangedSettings_NothingChanged verifies that changedSettings returns an
// empty map when no inputs have been modified.
func TestChangedSettings_NothingChanged(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	vals := &client.IndexSettingsValues{
		NumberOfReplicas: "1",
		RefreshInterval:  "5s",
	}
	m.applySettings(vals)

	changed := m.changedSettings()
	assert.Empty(t, changed, "no changes should produce empty map")
}

// TestChangedSettings_OneChange verifies that changedSettings returns only the
// field whose value was modified.
func TestChangedSettings_OneChange(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	vals := &client.IndexSettingsValues{
		NumberOfReplicas: "1",
		RefreshInterval:  "5s",
	}
	m.applySettings(vals)

	// Mutate the replicas field directly.
	setFieldInputValue(&m, "index.number_of_replicas", "3")

	changed := m.changedSettings()
	require.Len(t, changed, 1, "only the modified field should appear")
	assert.Equal(t, "3", changed["index.number_of_replicas"])
}

// TestChangedSettings_MultipleChanges verifies that changedSettings captures
// all modified fields and excludes unchanged ones.
func TestChangedSettings_MultipleChanges(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	vals := &client.IndexSettingsValues{
		NumberOfReplicas: "1",
		RefreshInterval:  "5s",
	}
	m.applySettings(vals)

	setFieldInputValue(&m, "index.number_of_replicas", "2")
	setFieldInputValue(&m, "index.refresh_interval", "30s")

	changed := m.changedSettings()
	assert.Len(t, changed, 2)
	assert.Equal(t, "2", changed["index.number_of_replicas"])
	assert.Equal(t, "30s", changed["index.refresh_interval"])
}

// TestChangedSettings_EmptyStringSendsNull verifies that an empty-string new
// value is sent as nil (JSON null) to clear the setting on ES.
func TestChangedSettings_EmptyStringSendsNull(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	vals := &client.IndexSettingsValues{
		Routing: client.IndexRoutingSettings{
			Allocation: client.IndexAllocationSettings{
				Include: client.IndexAllocationFilter{Name: "node-1"},
			},
		},
	}
	m.applySettings(vals)

	// Clear the field value (empty string ≠ "node-1").
	setFieldInputValue(&m, "index.routing.allocation.include._name", "")

	changed := m.changedSettings()
	require.Contains(t, changed, "index.routing.allocation.include._name")
	assert.Nil(t, changed["index.routing.allocation.include._name"], "empty string must be sent as nil (JSON null) to clear the ES setting")
}

// TestSettingsFormUpdate_DownNavigates verifies that the down arrow moves
// focusedField to the next field.
func TestSettingsFormUpdate_DownNavigates(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	m.loading = false // must not be loading for navigation to work

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m2.focusedField)
}

// TestSettingsFormUpdate_UpNavigates verifies that the up arrow moves
// focusedField to the previous field.
func TestSettingsFormUpdate_UpNavigates(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	m.loading = false
	m.focusedField = 2
	m.fields[0].input.Blur()
	m.fields[2].input.Focus()

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, m2.focusedField)
}

// TestSettingsFormUpdate_DownWrapsAtEnd verifies that pressing down on the
// last field wraps back to the first.
func TestSettingsFormUpdate_DownWrapsAtEnd(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	m.loading = false
	last := len(m.fields) - 1
	m.fields[0].input.Blur()
	m.focusedField = last
	m.fields[last].input.Focus()

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, m2.focusedField, "down on last field must wrap to 0")
}

// TestSettingsFormUpdate_UpWrapsAtStart verifies that pressing up on the first
// field wraps to the last field.
func TestSettingsFormUpdate_UpWrapsAtStart(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	m.loading = false // focusedField=0 by default

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, len(m.fields)-1, m2.focusedField, "up on first field must wrap to last")
}

// TestSettingsFormUpdate_TabNavigatesDown verifies that Tab behaves like down.
func TestSettingsFormUpdate_TabNavigatesDown(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	m.loading = false

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, m2.focusedField)
}

// TestSettingsFormUpdate_ShiftTabNavigatesUp verifies that Shift+Tab behaves like up.
func TestSettingsFormUpdate_ShiftTabNavigatesUp(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	m.loading = false
	m.fields[0].input.Blur()
	m.focusedField = 3
	m.fields[3].input.Focus()

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, 2, m2.focusedField)
}

// TestSettingsFormUpdate_CtrlS_SetsSubmittedFlag verifies that ctrl+s sets
// the submitted flag and returns no command.
func TestSettingsFormUpdate_CtrlS_SetsSubmittedFlag(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	m.loading = false

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	assert.True(t, m2.submitted, "ctrl+s must set submitted=true")
	assert.Nil(t, cmd, "ctrl+s must return nil cmd")
}

// TestSettingsFormUpdate_Esc_SetsCancelledFlag verifies that esc sets
// the cancelled flag and returns no command.
func TestSettingsFormUpdate_Esc_SetsCancelledFlag(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.True(t, m2.cancelled, "esc must set cancelled=true")
	assert.Nil(t, cmd, "esc must return nil cmd")
}

// TestSettingsFormUpdate_Esc_WhileLoading verifies that esc works even when
// the form is in loading state.
func TestSettingsFormUpdate_Esc_WhileLoading(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	assert.True(t, m.loading)

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.True(t, m2.cancelled, "esc while loading must set cancelled=true")
	assert.Nil(t, cmd, "esc while loading must return nil cmd")
}

// TestSettingsFormUpdate_CtrlS_BlockedWhileLoading verifies that ctrl+s is a
// no-op when the form is still loading settings from ES.
func TestSettingsFormUpdate_CtrlS_BlockedWhileLoading(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	assert.True(t, m.loading)

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	assert.Equal(t, 0, m2.focusedField, "focusedField must be unchanged while loading")
	assert.False(t, m2.submitted, "ctrl+s while loading must not set submitted flag")
	assert.Nil(t, cmd, "ctrl+s while loading must return nil cmd")
}

// TestSettingsFormUpdate_NavigationBlockedWhileLoading verifies that navigation
// keys are ignored when the form is in loading state.
func TestSettingsFormUpdate_NavigationBlockedWhileLoading(t *testing.T) {
	m := buildSettingsForm([]string{"idx"}, nil, nil)
	assert.True(t, m.loading)

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, m2.focusedField, "navigation must not change field while loading")
	assert.Nil(t, cmd)
}

// findField is a test helper that returns a pointer to the settingsField with
// the given ESKey, or nil if not found.
func findField(m SettingsFormModel, esKey string) *settingsField {
	for i := range m.fields {
		if m.fields[i].ESKey == esKey {
			return &m.fields[i]
		}
	}
	return nil
}

// setFieldInputValue is a test helper that directly sets the input value of
// the field with the given ESKey.
func setFieldInputValue(m *SettingsFormModel, esKey, val string) {
	for i := range m.fields {
		if m.fields[i].ESKey == esKey {
			m.fields[i].input.SetValue(val)
			return
		}
	}
}
