package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap holds all key bindings for the TUI.
type keyMap struct {
	Quit       key.Binding
	Refresh    key.Binding
	Tab        key.Binding
	ShiftTab   key.Binding
	Search     key.Binding
	Escape     key.Binding
	Help       key.Binding
	SortCol1   key.Binding
	SortCol2   key.Binding
	SortCol3   key.Binding
	SortCol4   key.Binding
	SortCol5   key.Binding
	SortCol6   key.Binding
	SortCol7   key.Binding
	SortCol8   key.Binding
	SortCol9   key.Binding
	PrevPage   key.Binding
	NextPage   key.Binding
}

// keys is the global key map.
var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh now"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next table"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev table"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	SortCol1: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "sort col 1")),
	SortCol2: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "sort col 2")),
	SortCol3: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "sort col 3")),
	SortCol4: key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "sort col 4")),
	SortCol5: key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "sort col 5")),
	SortCol6: key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "sort col 6")),
	SortCol7: key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "sort col 7")),
	SortCol8: key.NewBinding(key.WithKeys("8"), key.WithHelp("8", "sort col 8")),
	SortCol9: key.NewBinding(key.WithKeys("9"), key.WithHelp("9", "sort col 9")),
	PrevPage: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "prev page"),
	),
	NextPage: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next page"),
	),
}

// helpText is the full help string displayed in the footer when help is toggled on.
const helpText = "q/ctrl+c: quit  r: refresh  ?: toggle help"
