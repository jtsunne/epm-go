package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dm/epm-go/internal/model"
)

func TestIndexTableSetData_AppliesDefaultSort(t *testing.T) {
	m := NewIndexTable()
	rows := []model.IndexRow{
		{Name: "alpha", IndexingRate: 10.0},
		{Name: "gamma", IndexingRate: 300.0},
		{Name: "beta", IndexingRate: 50.0},
	}
	m.SetData(rows)

	require.Len(t, m.displayRows, 3)
	assert.Equal(t, "gamma", m.displayRows[0].Name, "highest IndexingRate should be first")
	assert.Equal(t, "beta", m.displayRows[1].Name)
	assert.Equal(t, "alpha", m.displayRows[2].Name, "lowest IndexingRate should be last")
}

func TestIndexTableSearch(t *testing.T) {
	m := NewIndexTable()
	m.search = "logs"
	rows := []model.IndexRow{
		{Name: "logs-2024", IndexingRate: 100.0},
		{Name: "metrics-daily", IndexingRate: 200.0},
		{Name: "logs-2023", IndexingRate: 50.0},
	}
	m.SetData(rows)

	require.Len(t, m.displayRows, 2, "only log indices should remain after filter")
	assert.Equal(t, "logs-2024", m.displayRows[0].Name, "higher rate first within filtered set")
	assert.Equal(t, "logs-2023", m.displayRows[1].Name)
}

func TestIndexTablePagination(t *testing.T) {
	m := NewIndexTable()
	rows := make([]model.IndexRow, 25)
	for i := range rows {
		rows[i] = model.IndexRow{
			Name:         fmt.Sprintf("index-%02d", i),
			IndexingRate: float64(i),
		}
	}
	m.SetData(rows)

	assert.Equal(t, 25, len(m.displayRows))
	assert.Equal(t, 3, pageCount(len(m.displayRows), m.pageSize))
}

// TestIndexTableNextPage_ClampsAtLastPage verifies that pressing → past the
// last page does not advance the page counter beyond pageCount-1.
func TestIndexTableNextPage_ClampsAtLastPage(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	rows := make([]model.IndexRow, 25) // 3 pages at pageSize=10
	for i := range rows {
		rows[i] = model.IndexRow{Name: fmt.Sprintf("index-%02d", i), IndexingRate: float64(i)}
	}
	m.SetData(rows)
	require.Equal(t, 3, pageCount(len(m.displayRows), m.pageSize))

	// Advance to last page.
	nextPage := tea.KeyMsg{Type: tea.KeyRight}
	m, _ = m.Update(nextPage)
	m, _ = m.Update(nextPage)
	assert.Equal(t, 2, m.page, "should be on page 2 (0-indexed last page)")

	// Press → again; must stay at page 2.
	m, _ = m.Update(nextPage)
	assert.Equal(t, 2, m.page, "page must not exceed last valid page index")
}

// TestIndexTableEscape_NoPageReset verifies that pressing Escape when there
// is no active search filter does not reset the page counter.
func TestIndexTableEscape_NoPageReset(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	rows := make([]model.IndexRow, 25)
	for i := range rows {
		rows[i] = model.IndexRow{Name: fmt.Sprintf("index-%02d", i), IndexingRate: float64(i)}
	}
	m.SetData(rows)

	// Navigate to page 1.
	nextPage := tea.KeyMsg{Type: tea.KeyRight}
	m, _ = m.Update(nextPage)
	require.Equal(t, 1, m.page)

	// Press Escape with no active search; page must not reset.
	esc := tea.KeyMsg{Type: tea.KeyEscape}
	m, _ = m.Update(esc)
	assert.Equal(t, 1, m.page, "Escape with no active filter must not reset page")
}

// TestIndexTableSort_NameAscendingByDefault verifies that pressing "1" (Name
// column) sorts ascending on first press, per the plan spec.
func TestIndexTableSort_NameAscendingByDefault(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	rows := []model.IndexRow{
		{Name: "zebra", IndexingRate: 100.0},
		{Name: "alpha", IndexingRate: 200.0},
		{Name: "mango", IndexingRate: 50.0},
	}
	m.SetData(rows)

	// Press "1" → Name column; default direction must be ascending.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	require.Len(t, m.displayRows, 3)
	assert.Equal(t, "alpha", m.displayRows[0].Name, "Name column should sort ascending on first press")
	assert.Equal(t, "mango", m.displayRows[1].Name)
	assert.Equal(t, "zebra", m.displayRows[2].Name)
}

func TestIndexTableRender_ContainsIndexName(t *testing.T) {
	m := NewIndexTable()
	rows := []model.IndexRow{
		{Name: "my-test-index", IndexingRate: 42.0},
	}
	m.SetData(rows)

	out := m.renderTable(nil)
	assert.True(t, strings.Contains(out, "my-test-index"), "rendered output should contain the index name")
}
