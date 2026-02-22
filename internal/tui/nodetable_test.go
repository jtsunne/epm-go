package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dm/epm-go/internal/model"
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
