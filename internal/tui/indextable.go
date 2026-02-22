package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dm/epm-go/internal/format"
	"github.com/dm/epm-go/internal/model"
)

// IndexTableModel is a sortable, paginated, searchable table of index statistics.
type IndexTableModel struct {
	tableModel
	allRows     []model.IndexRow // unfiltered source data
	displayRows []model.IndexRow // after filter + sort applied
}

// NewIndexTable returns an IndexTableModel with 9-column layout and
// default sort by IndexingRate (col 5) descending.
func NewIndexTable() IndexTableModel {
	cols := []columnDef{
		{Title: "Index Name", Width: 25, Align: "left",   Key: "name",       SortDesc: false},
		{Title: "P/T",        Width: 7,  Align: "center", Key: "shards",     SortDesc: true},
		{Title: "Total Size", Width: 10, Align: "right",  Key: "size",       SortDesc: true},
		{Title: "Shard Size", Width: 10, Align: "right",  Key: "shard_size", SortDesc: true},
		{Title: "Doc Count",  Width: 12, Align: "right",  Key: "docs",       SortDesc: true},
		{Title: "Idx/s",      Width: 8,  Align: "right",  Key: "idx_rate",   SortDesc: true},
		{Title: "Srch/s",     Width: 8,  Align: "right",  Key: "srch_rate",  SortDesc: true},
		{Title: "Idx Lat",    Width: 9,  Align: "right",  Key: "idx_lat",    SortDesc: true},
		{Title: "Srch Lat",   Width: 9,  Align: "right",  Key: "srch_lat",   SortDesc: true},
	}
	m := IndexTableModel{
		tableModel: newTableModel(cols),
	}
	m.sortCol = 5  // IndexingRate
	m.sortDesc = true
	return m
}

// SetData applies the current search filter and sort to rows, storing the
// result as displayRows ready for rendering.
func (m *IndexTableModel) SetData(rows []model.IndexRow) {
	m.allRows = rows
	filtered := filterIndexRows(m.allRows, m.search)
	m.displayRows = sortIndexRows(filtered, m.sortCol, m.sortDesc)
	m.clampPage(len(m.displayRows))
	m.clampCursor(m.currentPageRowCount(len(m.displayRows)))
}

// Update handles keyboard events for sorting, pagination, and search. It
// delegates to the embedded tableModel and re-applies filter/sort when the
// sort column, direction, or search term changes.
func (m IndexTableModel) Update(msg tea.Msg) (IndexTableModel, tea.Cmd) {
	prevSort := m.sortCol
	prevDesc := m.sortDesc
	prevSearch := m.search

	base, cmd := m.tableModel.Update(msg)
	m.tableModel = base

	if m.sortCol != prevSort || m.sortDesc != prevDesc || m.search != prevSearch {
		filtered := filterIndexRows(m.allRows, m.search)
		m.displayRows = sortIndexRows(filtered, m.sortCol, m.sortDesc)
	}
	m.clampPage(len(m.displayRows)) // always clamp after any key (e.g. NextPage)
	m.clampCursor(m.currentPageRowCount(len(m.displayRows)))
	return m, cmd
}

// renderTable renders the complete "Index Statistics" section: a header bar
// followed by the lipgloss table body for the current page.
func (m *IndexTableModel) renderTable(app *App) string {
	pc := pageCount(len(m.displayRows), m.pageSize)
	hdr := m.renderHeader("Index Statistics", m.page+1, pc, m.searching, m.search)

	// Compute proportional column widths for the current terminal width.
	// Padding headers to these widths guides the table's natural column layout
	// toward our preferred proportions rather than the library's even distribution.
	var colWidths []int
	if app != nil && app.width > 0 {
		colWidths = columnWidths(app.width, m.columns)
	}

	// Build column header strings, appending a sort direction arrow to the
	// active sort column.
	headers := make([]string, len(m.columns))
	for i, c := range m.columns {
		if i == m.sortCol {
			arrow := "↓"
			if !m.sortDesc {
				arrow = "↑"
			}
			headers[i] = c.Title + arrow
		} else {
			headers[i] = c.Title
		}
	}

	// Pad headers to target column widths so the table allocates proportional space.
	if len(colWidths) == len(m.columns) {
		for i, h := range headers {
			runes := []rune(h)
			if len(runes) < colWidths[i] {
				headers[i] = h + strings.Repeat(" ", colWidths[i]-len(runes))
			}
		}
	}

	// Determine which rows to display on the current page.
	allIdx := make([]int, len(m.displayRows))
	for i := range m.displayRows {
		allIdx[i] = i
	}
	pageIdx := currentPageIndices(allIdx, m.page, m.pageSize)

	if len(pageIdx) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, hdr, StyleDim.Render("  (no indices)"))
	}

	sortCol := m.sortCol
	t := ltable.New().
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == ltable.HeaderRow {
				if col == sortCol {
					return lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
				}
				return lipgloss.NewStyle().Bold(true).Foreground(colorGray)
			}
			base := lipgloss.NewStyle()
			if row%2 == 0 {
				base = base.Background(colorAlt)
			}
			switch col {
			case 5:
				return base.Foreground(colorGreen)
			case 6:
				return base.Foreground(colorCyan)
			case 7:
				return base.Foreground(colorPurple)
			case 8:
				return base.Foreground(colorOrange)
			default:
				return base.Foreground(colorWhite)
			}
		}).
		BorderStyle(lipgloss.NewStyle().Foreground(colorGray)).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderHeader(true).
		BorderColumn(false)

	if app != nil && app.width > 0 {
		t = t.Width(app.width)
	}

	for _, idx := range pageIdx {
		r := m.displayRows[idx]
		cells := make([]string, len(m.columns))
		for col := range m.columns {
			cells[col] = indexCellValue(r, col)
		}
		// Prevent cell wrapping: truncate name to allocated column width.
		if len(colWidths) > 0 && colWidths[0] > 0 {
			cells[0] = truncateName(cells[0], colWidths[0])
		}
		t = t.Row(cells...)
	}

	return lipgloss.JoinVertical(lipgloss.Left, hdr, t.String())
}

// renderHeader renders the title bar with search/sort/page hints.
// When searching is true, the live textinput view is shown instead of hints.
// When searchTerm is non-empty, the active filter is shown alongside the page info.
func (m *IndexTableModel) renderHeader(title string, page, pageCount int, searching bool, searchTerm string) string {
	pageInfo := fmt.Sprintf("Page %d/%d", page, pageCount)

	var right string
	switch {
	case searching:
		right = "Search: " + m.input.View()
	case searchTerm != "":
		right = fmt.Sprintf("filter=%q  %s", searchTerm, pageInfo)
	default:
		right = fmt.Sprintf("[/: search]  [1-9: sort]  [←→: page]  %s", pageInfo)
	}

	return StyleDim.Render(title + "  " + right)
}

// indexCellValue formats an IndexRow field for a given column index.
func indexCellValue(r model.IndexRow, col int) string {
	switch col {
	case 0:
		return sanitize(r.Name)
	case 1:
		return fmt.Sprintf("%d/%d", r.PrimaryShards, r.TotalShards)
	case 2:
		return format.FormatBytes(r.TotalSizeBytes)
	case 3:
		return format.FormatBytes(r.AvgShardSize)
	case 4:
		return format.FormatNumber(r.DocCount)
	case 5:
		return format.FormatRate(r.IndexingRate)
	case 6:
		return format.FormatRate(r.SearchRate)
	case 7:
		return format.FormatLatency(r.IndexLatency)
	case 8:
		return format.FormatLatency(r.SearchLatency)
	default:
		return ""
	}
}
