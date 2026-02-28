package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jtsunne/epm-go/internal/format"
	"github.com/jtsunne/epm-go/internal/model"
)

// NodeTableModel is a sortable, paginated, searchable table of node statistics.
type NodeTableModel struct {
	tableModel
	allRows     []model.NodeRow // unfiltered source data
	displayRows []model.NodeRow // after filter + sort applied
}

// NewNodeTable returns a NodeTableModel with 9-column layout and
// default sort by IndexingRate (col 3) descending.
func NewNodeTable() NodeTableModel {
	cols := []columnDef{
		{Title: "Node Name", Width: 20, SortDesc: false},
		{Title: "Role",      Width: 6,  SortDesc: false},
		{Title: "IP",        Width: 15, SortDesc: false},
		{Title: "Idx/s",     Width: 8,  SortDesc: true},
		{Title: "Srch/s",    Width: 8,  SortDesc: true},
		{Title: "Idx Lat",   Width: 9,  SortDesc: true},
		{Title: "Srch Lat",  Width: 9,  SortDesc: true},
		{Title: "Shards",    Width: 7,  SortDesc: true},
		{Title: "Disk%",     Width: 7,  SortDesc: true},
	}
	m := NodeTableModel{
		tableModel: newTableModel(cols),
	}
	m.sortCol = 3  // IndexingRate
	m.sortDesc = true
	return m
}

// SetData applies the current search filter and sort to rows, storing the
// result as displayRows ready for rendering.
func (m *NodeTableModel) SetData(rows []model.NodeRow) {
	m.allRows = rows
	filtered := filterNodeRows(m.allRows, m.search)
	m.displayRows = sortNodeRows(filtered, m.sortCol, m.sortDesc)
	m.clampPage(len(m.displayRows))
	m.clampCursor(m.currentPageRowCount(len(m.displayRows)))
}

// Update handles keyboard events for sorting, pagination, and search. It
// delegates to the embedded tableModel and re-applies filter/sort when the
// sort column, direction, or search term changes.
func (m NodeTableModel) Update(msg tea.Msg) (NodeTableModel, tea.Cmd) {
	prevSort := m.sortCol
	prevDesc := m.sortDesc
	prevSearch := m.search

	base, cmd := m.tableModel.Update(msg)
	m.tableModel = base

	if m.sortCol != prevSort || m.sortDesc != prevDesc || m.search != prevSearch {
		filtered := filterNodeRows(m.allRows, m.search)
		m.displayRows = sortNodeRows(filtered, m.sortCol, m.sortDesc)
	}
	m.clampPage(len(m.displayRows)) // always clamp after any key (e.g. NextPage)
	m.clampCursor(m.currentPageRowCount(len(m.displayRows)))
	return m, cmd
}

// renderTable renders the complete "Node Statistics" section: a header bar
// followed by the lipgloss table body for the current page.
func (m *NodeTableModel) renderTable(app *App) string {
	pc := pageCount(len(m.displayRows), m.pageSize)
	hdr := m.renderHeader("Node Statistics", m.page+1, pc, m.searching, m.search)

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
		return lipgloss.JoinVertical(lipgloss.Left, hdr, StyleDim.Render("  (no nodes)"))
	}

	sortCol := m.sortCol
	focused := m.focused
	cursor := m.cursor
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
			if focused && row == cursor {
				base = base.Background(colorSelectedBg)
			} else if row%2 == 0 {
				base = base.Background(colorAlt)
			}
			switch col {
			case 1:
				return base.Foreground(colorBlue)
			case 3:
				return base.Foreground(colorGreen)
			case 4:
				return base.Foreground(colorCyan)
			case 5:
				return base.Foreground(colorPurple)
			case 6:
				return base.Foreground(colorOrange)
			case 7:
				return base.Foreground(colorWhite)
			case 8:
				return base.Foreground(colorDiskYellow)
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
			cells[col] = nodeCellValue(r, col)
		}
		// Prevent cell wrapping: truncate name to allocated column width.
		if len(colWidths) > 0 && colWidths[0] > 0 {
			cells[0] = truncateName(cells[0], colWidths[0])
		}
		t = t.Row(cells...)
	}

	// Detail line: show full untruncated Name + Role + IP for the selected row when focused.
	var detailLine string
	if m.focused && len(pageIdx) > 0 && m.cursor < len(pageIdx) {
		r := m.displayRows[pageIdx[m.cursor]]
		detailLine = StyleDim.Render("  " + sanitize(r.Name) + "  " + sanitize(r.Role) + "  " + sanitize(r.IP))
	}
	if detailLine != "" {
		return lipgloss.JoinVertical(lipgloss.Left, hdr, t.String(), detailLine)
	}
	return lipgloss.JoinVertical(lipgloss.Left, hdr, t.String())
}

// renderHeader renders the title bar with search/sort/page hints.
// When searching is true, the live textinput view is shown instead of hints.
// When searchTerm is non-empty, the active filter is shown alongside the page info.
func (m *NodeTableModel) renderHeader(title string, page, pageCount int, searching bool, searchTerm string) string {
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

// nodeCellValue formats a NodeRow field for a given column index.
func nodeCellValue(r model.NodeRow, col int) string {
	switch col {
	case 0:
		return sanitize(r.Name)
	case 1:
		return sanitize(abbreviateRole(r.Role))
	case 2:
		return sanitize(r.IP)
	case 3:
		return format.FormatRate(r.IndexingRate)
	case 4:
		return format.FormatRate(r.SearchRate)
	case 5:
		return format.FormatLatency(r.IndexLatency)
	case 6:
		return format.FormatLatency(r.SearchLatency)
	case 7:
		if r.Shards < 0 {
			return "---"
		}
		return strconv.Itoa(r.Shards)
	case 8:
		if r.DiskPercent < 0 {
			return "---"
		}
		return format.FormatPercent(r.DiskPercent)
	default:
		return ""
	}
}

// abbreviateRole returns a short label for an Elasticsearch node role string.
// Common role strings: "master", "data", "ingest", "coordinating", "dimr", etc.
// Unknown roles are returned as-is (truncated to 6 chars).
func abbreviateRole(role string) string {
	// ES reports roles as a concatenation of abbreviation letters, e.g. "dimr".
	// Return it directly if it already looks abbreviated (≤6 chars).
	runes := []rune(role)
	if len(runes) <= 6 {
		return role
	}
	// Truncate long role strings to fit the column.
	return string(runes[:6])
}
