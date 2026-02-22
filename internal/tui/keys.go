package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap holds all key bindings for the TUI.
type keyMap struct {
	Quit     key.Binding
	Refresh  key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Search   key.Binding
	Escape   key.Binding
	Help     key.Binding
	PrevPage key.Binding
	NextPage key.Binding
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
const helpText = "tab: switch table  /: search  1-9: sort col  ←→: pages  r: refresh  q: quit  ?: close help"
