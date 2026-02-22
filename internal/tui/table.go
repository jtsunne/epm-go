package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// columnDef describes a single column in a table.
type columnDef struct {
	Title    string
	Width    int
	Align    string // "left", "right", "center"
	Key      string // sort key (informational)
	SortDesc bool   // default sort direction when first selected: true=descending (numeric), false=ascending (text)
}

// tableModel is the generic base for sortable, paginated, searchable tables.
type tableModel struct {
	columns   []columnDef
	sortCol   int // -1 = unsorted
	sortDesc  bool
	page      int // 0-indexed
	pageSize  int // default 10
	cursor    int // 0-indexed row within current page
	search    string
	searching bool
	input     textinput.Model
	focused   bool
}

// newTableModel initialises a tableModel with sensible defaults.
func newTableModel(cols []columnDef) tableModel {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 80
	return tableModel{
		columns:  cols,
		sortCol:  -1,
		pageSize: 10,
		input:    ti,
	}
}

// Update handles keyboard input for sorting, pagination, and search.
func (t tableModel) Update(msg tea.Msg) (tableModel, tea.Cmd) {
	if !t.focused {
		return t, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if t.searching {
			switch {
			case key.Matches(msg, keys.Escape):
				t.searching = false
				t.input.Blur()
				if t.input.Value() == "" {
					t.search = ""
					t.cursor = 0
				}
				return t, nil
			case msg.String() == "enter":
				t.search = t.input.Value()
				t.searching = false
				t.input.Blur()
				t.page = 0
				t.cursor = 0
				return t, nil
			default:
				var cmd tea.Cmd
				t.input, cmd = t.input.Update(msg)
				return t, cmd
			}
		}

		// Not searching — handle navigation keys.
		switch {
		case key.Matches(msg, keys.Search):
			t.searching = true
			t.input.SetValue(t.search)
			t.input.Focus()
			return t, textinput.Blink
		case key.Matches(msg, keys.Escape):
			if t.search != "" {
				t.search = ""
				t.input.SetValue("")
				t.page = 0
				t.cursor = 0
			}
			return t, nil
		case key.Matches(msg, keys.PrevPage):
			if t.page > 0 {
				t.page--
				t.cursor = 0
			}
			return t, nil
		case key.Matches(msg, keys.NextPage):
			t.page++
			t.cursor = 0
			return t, nil
		case key.Matches(msg, keys.CursorUp):
			if t.cursor > 0 {
				t.cursor--
			}
			return t, nil
		case key.Matches(msg, keys.CursorDown):
			t.cursor++
			return t, nil
		default:
			// Digit keys 1-9 → set sort column.
			col := digitToCol(msg.String())
			if col >= 0 && col < len(t.columns) {
				if col == t.sortCol {
					t.sortDesc = !t.sortDesc
				} else {
					t.sortCol = col
					// Use per-column default direction: descending for numeric
					// columns, ascending for text columns (name, role, IP).
					t.sortDesc = t.columns[col].SortDesc
				}
				t.page = 0
				t.cursor = 0
				return t, nil
			}
		}
	}
	return t, nil
}

// digitToCol converts a "1"–"9" key string to a 0-indexed column number.
// Returns -1 for any other string.
func digitToCol(s string) int {
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		return int(s[0]-'1')
	}
	return -1
}

// pageCount returns the total number of pages for totalRows rows at pageSize rows per page.
// Always at least 1.
func pageCount(totalRows, pageSize int) int {
	if totalRows == 0 || pageSize <= 0 {
		return 1
	}
	c := totalRows / pageSize
	if totalRows%pageSize != 0 {
		c++
	}
	return c
}

// currentPageIndices returns the slice of row indices visible on the current page.
// allIndices is typically [0, 1, 2, ... n-1] or a pre-filtered subset.
func currentPageIndices(allIndices []int, page, pageSize int) []int {
	if pageSize <= 0 || len(allIndices) == 0 {
		return allIndices
	}
	start := page * pageSize
	if start >= len(allIndices) {
		start = 0
	}
	end := start + pageSize
	if end > len(allIndices) {
		end = len(allIndices)
	}
	return allIndices[start:end]
}

// clampPage ensures the page index stays within valid bounds given the total
// number of rows and the configured pageSize.
func (t *tableModel) clampPage(totalRows int) {
	pc := pageCount(totalRows, t.pageSize)
	if t.page >= pc {
		t.page = pc - 1
	}
	if t.page < 0 {
		t.page = 0
	}
}

// clampCursor clamps the cursor to the valid range [0, pageRowCount-1].
// If pageRowCount is 0 or negative, cursor is set to 0.
func (t *tableModel) clampCursor(pageRowCount int) {
	if pageRowCount <= 0 {
		t.cursor = 0
		return
	}
	if t.cursor >= pageRowCount {
		t.cursor = pageRowCount - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

// currentPageRowCount returns the number of rows visible on the current page
// given the total number of rows.
func (t *tableModel) currentPageRowCount(totalRows int) int {
	if totalRows == 0 || t.pageSize <= 0 {
		return 0
	}
	start := t.page * t.pageSize
	if start >= totalRows {
		start = 0
	}
	end := start + t.pageSize
	if end > totalRows {
		end = totalRows
	}
	return end - start
}

// truncateName truncates s to fit within maxWidth runes, appending "..."
// if truncated. Returns s unchanged if it fits. Uses []rune for correct
// Unicode handling. ES index names are ASCII in practice, but node names
// can be arbitrary hostnames.
func truncateName(s string, maxWidth int) string {
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		if maxWidth <= 0 {
			return ""
		}
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}

// columnWidths distributes available terminal width across table columns,
// proportional to each column's preferred Width in its columnDef.
// Each column receives at least minColWidth characters.
// If available is 0 or negative, preferred widths are returned unchanged.
func columnWidths(available int, defs []columnDef) []int {
	const minColWidth = 4
	n := len(defs)
	result := make([]int, n)
	if n == 0 {
		return result
	}
	if available <= 0 {
		for i, d := range defs {
			result[i] = d.Width
		}
		return result
	}

	// Sum preferred widths.
	total := 0
	for _, d := range defs {
		total += d.Width
	}

	if total <= 0 {
		each := available / n
		if each < minColWidth {
			each = minColWidth
		}
		for i := range result {
			result[i] = each
		}
		return result
	}

	// Scale proportionally, assigning remainder to the last column.
	assigned := 0
	for i, d := range defs {
		if i == n-1 {
			w := available - assigned
			if w < minColWidth {
				w = minColWidth
			}
			result[i] = w
		} else {
			w := available * d.Width / total
			if w < minColWidth {
				w = minColWidth
			}
			result[i] = w
			assigned += w
		}
	}
	return result
}
