package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette — ES Performance Monitor.
//
// Every color is adaptive: lipgloss picks Light or Dark based on the detected
// terminal background (overridable via --theme, see SetTheme). Light variants
// use darker (Tailwind 600/700) shades so accents keep contrast on white.
var (
	colorGreen      = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#10b981"}
	colorYellow     = lipgloss.AdaptiveColor{Light: "#b45309", Dark: "#f59e0b"}
	colorRed        = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#ef4444"}
	colorGray       = lipgloss.AdaptiveColor{Light: "#64748b", Dark: "#6b7280"}
	colorBlue       = lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#3b82f6"}
	colorCyan       = lipgloss.AdaptiveColor{Light: "#0e7490", Dark: "#06b6d4"}
	colorPurple     = lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#8b5cf6"}
	colorIndigo     = lipgloss.AdaptiveColor{Light: "#4f46e5", Dark: "#6366f1"}
	colorOrange     = lipgloss.AdaptiveColor{Light: "#c2410c", Dark: "#f97316"}
	colorDiskYellow = lipgloss.AdaptiveColor{Light: "#a16207", Dark: "#eab308"} // Disk% column — spec color

	colorFg       = lipgloss.AdaptiveColor{Light: "#0f172a", Dark: "#f8fafc"} // primary text
	colorFgMuted  = lipgloss.AdaptiveColor{Light: "#334155", Dark: "#cbd5e1"} // secondary text
	colorOnAccent = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#1e293b"} // text on status-colored background
	colorHeaderBg = lipgloss.AdaptiveColor{Light: "#e2e8f0", Dark: "#1e293b"}
	colorSurface  = lipgloss.AdaptiveColor{Light: "#f1f5f9", Dark: "#0f172a"} // overview cards, zebra rows
	colorCursorBg = lipgloss.AdaptiveColor{Light: "#bfdbfe", Dark: "#1e3a5f"} // cursor row highlight
	colorMarkedBg = lipgloss.AdaptiveColor{Light: "#c7d2fe", Dark: "#3730a3"} // multi-selected rows
)

// SetTheme selects the light or dark palette. "auto" (or "") detects the
// terminal background; detection can be wrong under tmux/screen or some SSH
// setups, hence the explicit overrides. Must be called before tea.NewProgram:
// detection queries the terminal, which would race with Bubble Tea's input reader.
func SetTheme(theme string) error {
	switch theme {
	case "", "auto":
		lipgloss.SetHasDarkBackground(lipgloss.HasDarkBackground())
	case "light":
		lipgloss.SetHasDarkBackground(false)
	case "dark":
		lipgloss.SetHasDarkBackground(true)
	default:
		return fmt.Errorf("invalid theme %q: must be auto, light, or dark", theme)
	}
	return nil
}

// Status styles — bold foreground, used for cluster health indicator.
var (
	StyleStatusGreen   = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	StyleStatusYellow  = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	StyleStatusRed     = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	StyleStatusUnknown = lipgloss.NewStyle().Foreground(colorGray)
)

// StyleHeader — full-width dark header bar.
var StyleHeader = lipgloss.NewStyle().
	Background(colorHeaderBg).
	Foreground(colorFg).
	Padding(0, 1)

// StyleOverviewCard — bordered card for the 7-stat overview bar.
var StyleOverviewCard = lipgloss.NewStyle().
	Background(colorSurface).
	Foreground(colorFg).
	Padding(0, 1).
	Margin(0).
	Align(lipgloss.Center)

// Table styles.
var (
	StyleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Underline(true).
				Foreground(colorGray)

	StyleTableRow = lipgloss.NewStyle().
			Foreground(colorFg)

	StyleTableRowAlt = lipgloss.NewStyle().
				Foreground(colorFgMuted)
)

// Utility styles.
var (
	StyleError = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	StyleDim   = lipgloss.NewStyle().Foreground(colorGray)
)

// Named color styles for table cell coloring.
var (
	StyleGreen  = lipgloss.NewStyle().Foreground(colorGreen)
	StyleYellow = lipgloss.NewStyle().Foreground(colorYellow)
	StyleOrange = lipgloss.NewStyle().Foreground(colorOrange)
	StyleBlue   = lipgloss.NewStyle().Foreground(colorBlue)
	StyleCyan   = lipgloss.NewStyle().Foreground(colorCyan)
	StylePurple = lipgloss.NewStyle().Foreground(colorPurple)
	StyleRed    = lipgloss.NewStyle().Foreground(colorRed)
)

// StatusStyle returns the appropriate bold+foreground style for a cluster health string.
// Accepts "green", "yellow", "red" (case-insensitive).
func StatusStyle(status string) lipgloss.Style {
	switch strings.ToLower(status) {
	case "green":
		return StyleStatusGreen
	case "yellow":
		return StyleStatusYellow
	case "red":
		return StyleStatusRed
	default:
		return StyleStatusUnknown
	}
}
