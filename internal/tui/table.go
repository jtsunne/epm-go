package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// columnDef describes a single column in a table.
type columnDef struct {
	Title string
	Width int
	Align string // "left", "right", "center"
	Key   string // sort key (informational)
}

// tableModel is the generic base for sortable, paginated, searchable tables.
type tableModel struct {
	columns   []columnDef
	sortCol   int // -1 = unsorted
	sortDesc  bool
	page      int // 0-indexed
	pageSize  int // default 10
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
				}
				return t, nil
			case msg.String() == "enter":
				t.search = t.input.Value()
				t.searching = false
				t.input.Blur()
				t.page = 0
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
			t.search = ""
			t.input.SetValue("")
			t.page = 0
			return t, nil
		case key.Matches(msg, keys.PrevPage):
			if t.page > 0 {
				t.page--
			}
			return t, nil
		case key.Matches(msg, keys.NextPage):
			t.page++
			return t, nil
		default:
			// Digit keys 1-9 → set sort column.
			col := digitToCol(msg.String())
			if col >= 0 {
				if col == t.sortCol {
					t.sortDesc = !t.sortDesc
				} else {
					t.sortCol = col
					t.sortDesc = true // default: descending for new column
				}
				t.page = 0
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
