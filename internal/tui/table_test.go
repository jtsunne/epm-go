package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
