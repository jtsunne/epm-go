package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtsunne/epm-go/internal/model"
)

func TestTruncateName(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		maxWidth int
		want     string
	}{
		{"empty string", "", 10, ""},
		{"fits exactly", "hello", 5, "hello"},
		{"fits shorter", "hi", 10, "hi"},
		{"one over", "hello!", 5, "he..."},
		{"long name", "logstash-production-2024.01.15-000042", 20, "logstash-producti..."},
		{"width 0", "abc", 0, ""},
		{"width 1", "abc", 1, "a"},
		{"width 2", "abc", 2, "ab"},
		{"width 3", "abcd", 3, "abc"},
		{"width 4", "abcde", 4, "a..."},
		{"unicode fits", "héllo", 5, "héllo"},
		{"unicode truncated", "héllo world", 8, "héllo..."},
		// Wide characters (CJK): each occupies 2 terminal columns.
		{"wide chars fit", "中文", 4, "中文"},
		{"wide chars truncated", "中文测试", 5, "中..."},
		{"wide chars truncated exact", "中文测试", 7, "中文..."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateName(tc.s, tc.maxWidth)
			assert.Equal(t, tc.want, got)
			// Result must never exceed maxWidth terminal display cells.
			if tc.maxWidth > 0 {
				assert.LessOrEqual(t, runewidth.StringWidth(got), tc.maxWidth,
					"result display width must not exceed maxWidth")
			}
		})
	}
}

func TestColumnWidths_ZeroAvailable(t *testing.T) {
	defs := []columnDef{
		{Title: "A", Width: 10},
		{Title: "B", Width: 20},
	}
	got := columnWidths(0, defs)
	assert.Equal(t, []int{10, 20}, got, "zero available → preferred widths returned unchanged")
}

func TestColumnWidths_NegativeAvailable(t *testing.T) {
	defs := []columnDef{
		{Title: "A", Width: 15},
	}
	got := columnWidths(-1, defs)
	assert.Equal(t, []int{15}, got)
}

func TestColumnWidths_EmptyDefs(t *testing.T) {
	got := columnWidths(100, nil)
	assert.Equal(t, []int{}, got)
}

func TestColumnWidths_ProportionalDistribution(t *testing.T) {
	// Two equal-preferred columns get equal share of available.
	defs := []columnDef{
		{Title: "A", Width: 10},
		{Title: "B", Width: 10},
	}
	got := columnWidths(100, defs)
	assert.Len(t, got, 2)
	assert.Equal(t, got[0]+got[1], 100, "widths must sum to available")
	assert.Equal(t, got[0], got[1], "equal preferred widths → equal distribution")
}

func TestColumnWidths_ProportionalUnequal(t *testing.T) {
	// Column A preferred 10, column B preferred 30 (1:3 ratio).
	// With available=80: A=20, B=60.
	defs := []columnDef{
		{Title: "A", Width: 10},
		{Title: "B", Width: 30},
	}
	got := columnWidths(80, defs)
	assert.Equal(t, 2, len(got))
	// A should get 80*10/40 = 20, B gets 80-20 = 60.
	assert.Equal(t, 20, got[0])
	assert.Equal(t, 60, got[1])
}

func TestColumnWidths_ClampsToMinimum(t *testing.T) {
	// Very narrow terminal: all columns clamped to minColWidth (4).
	defs := []columnDef{
		{Title: "A", Width: 10},
		{Title: "B", Width: 10},
		{Title: "C", Width: 10},
	}
	// With available=6: 6*10/30 = 2 per col → clamped to 4 each.
	// The last column gets remaining = 6 - 4 = 2, clamped to 4.
	got := columnWidths(6, defs)
	for i, w := range got {
		assert.GreaterOrEqual(t, w, 4, "column %d width must be >= 4", i)
	}
}

func TestColumnWidths_SingleColumn(t *testing.T) {
	defs := []columnDef{
		{Title: "Name", Width: 20},
	}
	got := columnWidths(50, defs)
	assert.Equal(t, []int{50}, got, "single column gets all available width")
}

func TestColumnWidths_ThreeColumns(t *testing.T) {
	// 3 columns with preferred widths 20, 10, 10 → total=40.
	// With available=80: A=40, B=20, C=remainder=20.
	defs := []columnDef{
		{Title: "A", Width: 20},
		{Title: "B", Width: 10},
		{Title: "C", Width: 10},
	}
	got := columnWidths(80, defs)
	assert.Equal(t, 3, len(got))
	assert.Equal(t, 80*20/40, got[0]) // 40
	assert.Equal(t, 80*10/40, got[1]) // 20
	// Last gets remainder: 80 - 40 - 20 = 20
	assert.Equal(t, 20, got[2])
}

func TestPageCount(t *testing.T) {
	tests := []struct {
		totalRows, pageSize, want int
	}{
		{0, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{20, 10, 2},
		{21, 10, 3},
		{5, 0, 1},
	}
	for _, tc := range tests {
		got := pageCount(tc.totalRows, tc.pageSize)
		assert.Equal(t, tc.want, got, "pageCount(%d, %d)", tc.totalRows, tc.pageSize)
	}
}

func TestCurrentPageIndices(t *testing.T) {
	all := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// First page.
	got := currentPageIndices(all, 0, 4)
	assert.Equal(t, []int{0, 1, 2, 3}, got)

	// Second page.
	got = currentPageIndices(all, 1, 4)
	assert.Equal(t, []int{4, 5, 6, 7}, got)

	// Page beyond range resets to start.
	got = currentPageIndices(all, 5, 4)
	assert.Equal(t, []int{0, 1, 2, 3}, got)

	// Empty slice.
	got = currentPageIndices(nil, 0, 4)
	assert.Nil(t, got)
}

// makeIndexRows returns n IndexRow values with IndexingRate = float64(i).
func makeIndexRows(n int) []model.IndexRow {
	rows := make([]model.IndexRow, n)
	for i := range rows {
		rows[i] = model.IndexRow{
			Name:         fmt.Sprintf("index-%02d", i),
			IndexingRate: float64(i),
		}
	}
	return rows
}

// TestTableModel_CursorDownUp verifies that cursor increments on Down and
// decrements on Up, and that it cannot go below 0.
func TestTableModel_CursorDownUp(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	m.SetData(makeIndexRows(5))
	require.Equal(t, 0, m.cursor, "cursor starts at 0")

	down := tea.KeyMsg{Type: tea.KeyDown}
	up := tea.KeyMsg{Type: tea.KeyUp}

	m, _ = m.Update(down)
	assert.Equal(t, 1, m.cursor, "cursor should move to 1")

	m, _ = m.Update(down)
	assert.Equal(t, 2, m.cursor, "cursor should move to 2")

	m, _ = m.Update(up)
	assert.Equal(t, 1, m.cursor, "cursor should move back to 1")

	m, _ = m.Update(up)
	assert.Equal(t, 0, m.cursor, "cursor should be at 0")

	// Cannot go below 0.
	m, _ = m.Update(up)
	assert.Equal(t, 0, m.cursor, "cursor must not go below 0")
}

// TestTableModel_CursorVimKeys verifies j/k work as aliases for down/up.
func TestTableModel_CursorVimKeys(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	m.SetData(makeIndexRows(5))

	j := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	k := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}

	m, _ = m.Update(j)
	assert.Equal(t, 1, m.cursor, "j moves cursor down")

	m, _ = m.Update(k)
	assert.Equal(t, 0, m.cursor, "k moves cursor up")
}

// TestTableModel_CursorClampedAtPageEnd verifies cursor is clamped to the last
// row in the page and cannot exceed it.
func TestTableModel_CursorClampedAtPageEnd(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	m.SetData(makeIndexRows(3)) // 3 rows, pageSize=10 → only 3 rows on page

	down := tea.KeyMsg{Type: tea.KeyDown}

	m, _ = m.Update(down)
	m, _ = m.Update(down)
	assert.Equal(t, 2, m.cursor, "cursor at last row (index 2)")

	// Down again: clamped at 2.
	m, _ = m.Update(down)
	assert.Equal(t, 2, m.cursor, "cursor clamped at last row")
}

// TestTableModel_CursorResetOnPageChange verifies cursor resets to 0 when
// navigating to previous or next page.
func TestTableModel_CursorResetOnPageChange(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	m.SetData(makeIndexRows(25)) // 3 pages at pageSize=10

	down := tea.KeyMsg{Type: tea.KeyDown}
	next := tea.KeyMsg{Type: tea.KeyRight}
	prev := tea.KeyMsg{Type: tea.KeyLeft}

	// Move cursor down on page 0.
	m, _ = m.Update(down)
	m, _ = m.Update(down)
	require.Equal(t, 2, m.cursor)

	// Go to next page → cursor resets.
	m, _ = m.Update(next)
	assert.Equal(t, 0, m.cursor, "cursor resets to 0 on next page")
	assert.Equal(t, 1, m.page)

	// Move cursor on page 1.
	m, _ = m.Update(down)
	require.Equal(t, 1, m.cursor)

	// Go back → cursor resets.
	m, _ = m.Update(prev)
	assert.Equal(t, 0, m.cursor, "cursor resets to 0 on prev page")
	assert.Equal(t, 0, m.page)
}

// TestTableModel_CursorResetOnSortChange verifies cursor resets to 0 when
// the sort column changes via digit key.
func TestTableModel_CursorResetOnSortChange(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	m.SetData(makeIndexRows(5))

	down := tea.KeyMsg{Type: tea.KeyDown}
	sort1 := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}

	m, _ = m.Update(down)
	m, _ = m.Update(down)
	require.Equal(t, 2, m.cursor)

	// Press sort key → cursor resets.
	m, _ = m.Update(sort1)
	assert.Equal(t, 0, m.cursor, "cursor resets to 0 on sort column change")
}

// TestTableModel_CursorResetOnSearchApply verifies cursor resets to 0 when
// a search is confirmed with Enter.
func TestTableModel_CursorResetOnSearchApply(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	rows := []model.IndexRow{
		{Name: "alpha", IndexingRate: 1},
		{Name: "beta", IndexingRate: 2},
		{Name: "gamma", IndexingRate: 3},
	}
	m.SetData(rows)

	down := tea.KeyMsg{Type: tea.KeyDown}
	searchKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	enter := tea.KeyMsg{Type: tea.KeyEnter}

	// Move cursor.
	m, _ = m.Update(down)
	require.Equal(t, 1, m.cursor)

	// Open search, type nothing, press Enter.
	m, _ = m.Update(searchKey)
	require.True(t, m.searching)
	m, _ = m.Update(enter)
	assert.Equal(t, 0, m.cursor, "cursor resets to 0 after search confirm")
}

// TestTableModel_CursorResetOnSearchClear verifies cursor resets to 0 when
// Escape clears an active search filter (non-searching mode).
func TestTableModel_CursorResetOnSearchClear(t *testing.T) {
	m := NewIndexTable()
	m.focused = true
	m.search = "alpha"
	rows := []model.IndexRow{
		{Name: "alpha-1", IndexingRate: 1},
		{Name: "alpha-2", IndexingRate: 2},
	}
	m.SetData(rows)

	down := tea.KeyMsg{Type: tea.KeyDown}
	esc := tea.KeyMsg{Type: tea.KeyEscape}

	m, _ = m.Update(down)
	require.Equal(t, 1, m.cursor)

	// Escape clears active search → cursor resets.
	m, _ = m.Update(esc)
	assert.Equal(t, 0, m.cursor, "cursor resets when Escape clears active search")
	assert.Equal(t, "", m.search, "search filter cleared")
}

// TestTableModel_ClampCursor verifies clampCursor bounds cursor correctly.
func TestTableModel_ClampCursor(t *testing.T) {
	base := newTableModel(nil)

	// cursor above page: clamped to last row.
	base.cursor = 10
	base.clampCursor(5)
	assert.Equal(t, 4, base.cursor, "cursor clamped to pageRowCount-1")

	// cursor below 0: clamped to 0.
	base.cursor = -1
	base.clampCursor(5)
	assert.Equal(t, 0, base.cursor)

	// zero page rows: cursor set to 0.
	base.cursor = 3
	base.clampCursor(0)
	assert.Equal(t, 0, base.cursor)
}
