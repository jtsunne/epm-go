package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtsunne/epm-go/internal/model"
)

func TestNodeTableSetData_SortByRate(t *testing.T) {
	m := NewNodeTable()
	rows := []model.NodeRow{
		{Name: "node-1", IP: "10.0.0.1", IndexingRate: 50.0},
		{Name: "node-2", IP: "10.0.0.2", IndexingRate: 200.0},
		{Name: "node-3", IP: "10.0.0.3", IndexingRate: 100.0},
	}
	m.SetData(rows)

	require.Len(t, m.displayRows, 3)
	assert.Equal(t, "node-2", m.displayRows[0].Name, "highest IndexingRate should be first")
	assert.Equal(t, "node-3", m.displayRows[1].Name)
	assert.Equal(t, "node-1", m.displayRows[2].Name, "lowest IndexingRate should be last")
}

func TestNodeTableSearch_ByIP(t *testing.T) {
	m := NewNodeTable()
	m.search = "192.168.1"
	rows := []model.NodeRow{
		{Name: "node-1", IP: "192.168.1.1", IndexingRate: 100.0},
		{Name: "node-2", IP: "10.0.0.1", IndexingRate: 200.0},
		{Name: "node-3", IP: "192.168.1.2", IndexingRate: 50.0},
	}
	m.SetData(rows)

	require.Len(t, m.displayRows, 2, "only nodes with matching IP should remain")
	assert.Equal(t, "node-1", m.displayRows[0].Name, "higher rate first within filtered set")
	assert.Equal(t, "node-3", m.displayRows[1].Name)
}

// TestNodeTableNextPage_ClampsAtLastPage verifies that pressing → past the
// last page does not advance the page counter beyond pageCount-1.
func TestNodeTableNextPage_ClampsAtLastPage(t *testing.T) {
	m := NewNodeTable()
	m.focused = true
	// 3 rows, pageSize=10 → 1 page; page must stay at 0
	rows := []model.NodeRow{
		{Name: "node-1", IP: "10.0.0.1", IndexingRate: 100.0},
		{Name: "node-2", IP: "10.0.0.2", IndexingRate: 200.0},
		{Name: "node-3", IP: "10.0.0.3", IndexingRate: 50.0},
	}
	m.SetData(rows)
	require.Equal(t, 0, m.page)

	// Press → three times; should stay at page 0 (only 1 page).
	nextPage := tea.KeyMsg{Type: tea.KeyRight}
	for i := 0; i < 3; i++ {
		m, _ = m.Update(nextPage)
	}
	assert.Equal(t, 0, m.page, "page must not exceed last valid page index")
}

// TestNodeTableSort_NameAscendingByDefault verifies that pressing "1" (Name
// column) sorts ascending on first press, per the plan spec.
func TestNodeTableSort_NameAscendingByDefault(t *testing.T) {
	m := NewNodeTable()
	m.focused = true
	rows := []model.NodeRow{
		{Name: "zebra", IP: "10.0.0.1", IndexingRate: 100.0},
		{Name: "alpha", IP: "10.0.0.2", IndexingRate: 200.0},
		{Name: "mango", IP: "10.0.0.3", IndexingRate: 50.0},
	}
	m.SetData(rows)

	// Press "1" to sort by Name column; default should be ascending.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	require.Len(t, m.displayRows, 3)
	assert.Equal(t, "alpha", m.displayRows[0].Name, "Name column should sort ascending on first press")
	assert.Equal(t, "mango", m.displayRows[1].Name)
	assert.Equal(t, "zebra", m.displayRows[2].Name)
}

// TestNodeTableDetailLine_FocusedShowsFullName verifies that when the table is
// focused, the rendered output contains the full Name, Role, and IP of the
// selected row in the detail line below the table body.
func TestNodeTableDetailLine_FocusedShowsFullName(t *testing.T) {
	m := NewNodeTable()
	m.focused = true
	rows := []model.NodeRow{
		{Name: "very-long-node-hostname-prod-001", Role: "dimr", IP: "192.168.100.50"},
	}
	m.SetData(rows)

	out := m.renderTable(nil)
	assert.True(t, strings.Contains(out, "very-long-node-hostname-prod-001"),
		"detail line should contain the full untruncated node name when focused")
	assert.True(t, strings.Contains(out, "192.168.100.50"),
		"detail line should contain the node IP when focused")
}

// TestNodeTableDetailLine_UnfocusedAbsent verifies that the focused table
// output is longer than the unfocused output, confirming the detail line is
// only rendered when the table is focused.
func TestNodeTableDetailLine_UnfocusedAbsent(t *testing.T) {
	rows := []model.NodeRow{
		{Name: "very-long-node-hostname-prod-001", Role: "dimr", IP: "192.168.100.50"},
	}

	mUnfocused := NewNodeTable()
	mUnfocused.focused = false
	mUnfocused.SetData(rows)
	outUnfocused := mUnfocused.renderTable(nil)

	mFocused := NewNodeTable()
	mFocused.focused = true
	mFocused.SetData(rows)
	outFocused := mFocused.renderTable(nil)

	assert.Greater(t, len(outFocused), len(outUnfocused),
		"focused table output should be longer than unfocused (has detail line)")
}

// TestNodeTableDetailLine_CursorNonZero verifies that the detail line shows
// the name of the row under the cursor, not always the first row.
func TestNodeTableDetailLine_CursorNonZero(t *testing.T) {
	m := NewNodeTable()
	m.focused = true
	rows := []model.NodeRow{
		{Name: "node-alpha", Role: "m", IP: "10.0.0.1", IndexingRate: 300.0},
		{Name: "node-beta", Role: "d", IP: "10.0.0.2", IndexingRate: 200.0},
		{Name: "node-gamma", Role: "d", IP: "10.0.0.3", IndexingRate: 100.0},
	}
	m.SetData(rows)
	// Default sort: by IndexingRate desc → node-alpha, node-beta, node-gamma.

	down := tea.KeyMsg{Type: tea.KeyDown}
	m, _ = m.Update(down)
	m, _ = m.Update(down)
	// cursor is now at row 2 → "node-gamma"

	out := m.renderTable(nil)
	assert.True(t, strings.Contains(out, "node-gamma"),
		"detail line should show the name of the row at cursor position 2")
}

// TestAbbreviateRole verifies that role strings are returned as-is when ≤6
// bytes, and truncated when longer.
func TestAbbreviateRole(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"dimr", "dimr"},
		{"master", "master"},
		{"coordinating", "coordi"},
		{"data", "data"},
		{"", ""},
		{"abcdef", "abcdef"},
		{"abcdefg", "abcdef"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, abbreviateRole(tt.input), "input=%q", tt.input)
	}
}
