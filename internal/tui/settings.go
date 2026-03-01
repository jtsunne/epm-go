package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jtsunne/epm-go/internal/client"
)

// settingsField holds the state for a single editable settings field.
type settingsField struct {
	Label       string
	ESKey       string // full dotted ES key, e.g. "index.number_of_replicas"
	currentVal  string
	suggestions []string
	input       textinput.Model
}

// SettingsFormModel manages the state of the index settings editor form.
type SettingsFormModel struct {
	fields       []settingsField
	focusedField int
	loading      bool
	loadErr      string
	names        []string // index names being edited
	submitted    bool     // set by ctrl+s; cleared by parent after handling
	cancelled    bool     // set by esc; cleared by parent after handling
}

// buildSettingsForm creates a SettingsFormModel with all 11 editable fields.
// nodeNames and nodeIPs are used as suggestions for routing allocation fields.
func buildSettingsForm(names []string, nodeNames []string, nodeIPs []string) SettingsFormModel {
	fields := []settingsField{
		{
			Label:       "Replicas",
			ESKey:       "index.number_of_replicas",
			suggestions: []string{"0", "1", "2", "3"},
		},
		{
			Label:       "Refresh Interval",
			ESKey:       "index.refresh_interval",
			suggestions: []string{"-1", "1s", "5s", "30s", "60s"},
		},
		{
			Label:       "Allocation Include Name",
			ESKey:       "index.routing.allocation.include._name",
			suggestions: nodeNames,
		},
		{
			Label:       "Allocation Exclude Name",
			ESKey:       "index.routing.allocation.exclude._name",
			suggestions: nodeNames,
		},
		{
			Label:       "Allocation Require Name",
			ESKey:       "index.routing.allocation.require._name",
			suggestions: nodeNames,
		},
		{
			Label:       "Allocation Include IP",
			ESKey:       "index.routing.allocation.include._ip",
			suggestions: nodeIPs,
		},
		{
			Label:       "Allocation Exclude IP",
			ESKey:       "index.routing.allocation.exclude._ip",
			suggestions: nodeIPs,
		},
		{
			Label:       "Allocation Require IP",
			ESKey:       "index.routing.allocation.require._ip",
			suggestions: nodeIPs,
		},
		{
			Label:       "Total Shards Per Node",
			ESKey:       "index.routing.allocation.total_shards_per_node",
			suggestions: []string{"-1", "1", "2", "5"},
		},
		{
			Label:       "Total Fields Limit",
			ESKey:       "index.mapping.total_fields.limit",
			suggestions: []string{"1000", "2000", "5000", "10000"},
		},
		{
			Label:       "Read-Only Allow Delete",
			ESKey:       "index.blocks.read_only_allow_delete",
			suggestions: []string{"true", "false", ""},
		},
	}

	for i := range fields {
		ti := textinput.New()
		ti.CharLimit = 256
		fields[i].input = ti
	}

	if len(fields) > 0 {
		fields[0].input.Focus()
	}

	return SettingsFormModel{
		fields:       fields,
		focusedField: 0,
		loading:      true,
		names:        names,
	}
}

// applySettings populates form fields with current values from ES settings
// and clears the loading state.
func (m *SettingsFormModel) applySettings(vals *client.IndexSettingsValues) {
	if vals == nil {
		m.loading = false
		return
	}
	valueMap := map[string]string{
		"index.number_of_replicas":                         vals.NumberOfReplicas,
		"index.refresh_interval":                           vals.RefreshInterval,
		"index.routing.allocation.include._name":           vals.Routing.Allocation.Include.Name,
		"index.routing.allocation.exclude._name":           vals.Routing.Allocation.Exclude.Name,
		"index.routing.allocation.require._name":           vals.Routing.Allocation.Require.Name,
		"index.routing.allocation.include._ip":             vals.Routing.Allocation.Include.IP,
		"index.routing.allocation.exclude._ip":             vals.Routing.Allocation.Exclude.IP,
		"index.routing.allocation.require._ip":             vals.Routing.Allocation.Require.IP,
		"index.routing.allocation.total_shards_per_node":  vals.Routing.Allocation.TotalShardsPerNode,
		"index.mapping.total_fields.limit":                 vals.Mapping.TotalFields.Limit,
		"index.blocks.read_only_allow_delete":              vals.Blocks.ReadOnlyAllowDelete,
	}
	for i := range m.fields {
		if val, ok := valueMap[m.fields[i].ESKey]; ok {
			m.fields[i].currentVal = val
			m.fields[i].input.SetValue(val)
		}
	}
	m.loading = false
}

// Update handles keyboard input for the settings form.
// If ctrl+s is pressed, sets m.submitted=true (parent checks this flag).
// If esc is pressed, sets m.cancelled=true (parent checks this flag).
// ↑/↓ and Tab/Shift+Tab navigate between fields.
// All other keys are forwarded to the focused field's text input.
func (m SettingsFormModel) Update(msg tea.Msg) (SettingsFormModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		if !m.loading && len(m.fields) > 0 {
			var cmd tea.Cmd
			m.fields[m.focusedField].input, cmd = m.fields[m.focusedField].input.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Esc always cancels regardless of loading state.
	if keyMsg.String() == "esc" {
		m.cancelled = true
		return m, nil
	}

	if m.loading {
		return m, nil
	}

	switch keyMsg.String() {
	case "ctrl+s":
		m.submitted = true
		return m, nil

	case "up", "shift+tab":
		m.fields[m.focusedField].input.Blur()
		if m.focusedField > 0 {
			m.focusedField--
		} else {
			m.focusedField = len(m.fields) - 1
		}
		m.fields[m.focusedField].input.Focus()
		return m, nil

	case "down", "tab":
		m.fields[m.focusedField].input.Blur()
		if m.focusedField < len(m.fields)-1 {
			m.focusedField++
		} else {
			m.focusedField = 0
		}
		m.fields[m.focusedField].input.Focus()
		return m, nil

	default:
		var cmd tea.Cmd
		m.fields[m.focusedField].input, cmd = m.fields[m.focusedField].input.Update(msg)
		return m, cmd
	}
}

// changedSettings returns a map of ESKey → new value for all fields whose
// current input value differs from the original value loaded from ES.
// Empty-string values are sent as nil (JSON null) to clear the setting on ES.
// Returns an empty map if nothing has changed.
func (m SettingsFormModel) changedSettings() map[string]any {
	result := make(map[string]any)
	for _, f := range m.fields {
		newVal := f.input.Value()
		if newVal != f.currentVal {
			if newVal == "" {
				result[f.ESKey] = nil
			} else {
				result[f.ESKey] = newVal
			}
		}
	}
	return result
}

// settingsLoadCmd fetches current settings for a single index and returns
// a SettingsLoadedMsg with the parsed values.
// nonce is embedded in the message so the App can discard stale responses.
func settingsLoadCmd(c client.ESClient, name string, nonce int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		vals, err := c.GetIndexSettings(ctx, name)
		return SettingsLoadedMsg{Values: vals, Err: err, Nonce: nonce}
	}
}

// settingsUpdateCmd sends a PUT /_settings request for the given index names
// with the supplied flat key→value map and returns a SettingsResultMsg.
// nonce is embedded in the message so the App can discard stale responses.
func settingsUpdateCmd(c client.ESClient, names []string, settings map[string]any, nonce int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := c.UpdateIndexSettings(ctx, names, settings)
		return SettingsResultMsg{Names: names, Err: err, Nonce: nonce}
	}
}

// renderSettingsForm renders the full-screen settings form overlay.
// The caller (View) renders the cluster header above and footer below.
func renderSettingsForm(app *App) string {
	width := app.width
	if width <= 0 {
		width = 80
	}
	height := app.height
	if height <= 0 {
		height = 24
	}

	form := &app.settingsForm

	// Build title bar (same style as renderDeleteConfirm).
	nameLabel := strings.Join(form.names, ", ")
	if len(form.names) > 1 {
		nameLabel = fmt.Sprintf("%d indices", len(form.names))
	}
	titleText := fmt.Sprintf("Edit Index Settings: %s", sanitize(nameLabel))
	hintText := StyleDim.Render("[ctrl+s: save  esc: cancel]")
	hintVW := lipgloss.Width(hintText)
	titleVW := lipgloss.Width(titleText)
	innerWidth := width - 2 // StyleHeader has Padding(0,1)
	gap := innerWidth - titleVW - hintVW
	if gap < 1 {
		gap = 1
	}
	titleRow := titleText + strings.Repeat(" ", gap) + hintText
	titleBar := StyleHeader.Width(width).MaxWidth(width).Render(titleRow)
	titleH := lipgloss.Height(titleBar)

	// When editing multiple indices, show which index values were pre-filled from.
	var subtitleBar string
	var subtitleH int
	if len(form.names) > 1 {
		subtitleText := StyleDim.Render(fmt.Sprintf("  pre-filled from: %s", sanitize(form.names[0])))
		subtitleBar = lipgloss.NewStyle().Width(width).Render(subtitleText)
		subtitleH = lipgloss.Height(subtitleBar)
	}

	headerH := renderedHeight(renderHeader(app))
	footerH := renderedHeight(renderFooter(app))
	availH := height - headerH - titleH - subtitleH - footerH
	if availH < 1 {
		availH = 1
	}

	// titlePrefix is prepended to all return paths below.
	titlePrefix := titleBar
	if subtitleBar != "" {
		titlePrefix = titleBar + "\n" + subtitleBar
	}


	// Loading state.
	if form.loading {
		line := "  Loading current settings..."
		lines := make([]string, availH)
		lines[0] = ""
		if availH > 1 {
			lines[1] = line
		}
		return titlePrefix + "\n" + strings.Join(lines, "\n")
	}

	// Error state.
	if form.loadErr != "" {
		line := "  " + StyleError.Render("Error: "+form.loadErr)
		lines := make([]string, availH)
		lines[0] = ""
		if availH > 1 {
			lines[1] = line
		}
		return titlePrefix + "\n" + strings.Join(lines, "\n")
	}

	// Compute line counts per field (label row + optional suggestions row).
	contentH := availH - 1 // reserve 1 line for top padding
	if contentH < 1 {
		contentH = 1
	}
	linesPerField := make([]int, len(form.fields))
	for i, f := range form.fields {
		if len(f.suggestions) > 0 {
			linesPerField[i] = 2
		} else {
			linesPerField[i] = 1
		}
	}

	// Scroll to keep focused field visible: walk backwards from focusedField,
	// accumulating used lines, to find the first field that still fits.
	firstVisible := form.focusedField
	usedLines := 0
	for i := form.focusedField; i >= 0; i-- {
		if usedLines+linesPerField[i] > contentH {
			break
		}
		usedLines += linesPerField[i]
		firstVisible = i
	}

	// Render fields starting from firstVisible. Each field: label+input row,
	// then optional suggestions row. We render up to availH lines total.
	var lines []string
	lines = append(lines, "") // top padding

	selectedBg := lipgloss.NewStyle().Background(colorSelectedBg)

	for i := firstVisible; i < len(form.fields); i++ {
		if len(lines) >= availH {
			break
		}
		f := form.fields[i]

		// Label column (fixed 28 chars) and input column.
		label := fmt.Sprintf("  %-28s", f.Label)
		inputView := f.input.View()

		// For the focused row, highlight the entire line.
		row := label + inputView
		if i == form.focusedField {
			row = selectedBg.Width(width - 2).Render(row)
		}
		lines = append(lines, row)

		// Suggestions line (dimmed, indented).
		if len(lines) < availH && len(f.suggestions) > 0 {
			sug := "  " + strings.Repeat(" ", 28) + StyleDim.Render("Suggestions: "+strings.Join(f.suggestions, "  "))
			lines = append(lines, sug)
		}
	}

	// Pad to availH.
	for len(lines) < availH {
		lines = append(lines, "")
	}

	return titlePrefix + "\n" + strings.Join(lines[:availH], "\n")
}
