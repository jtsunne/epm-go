package tui

import (
	"fmt"
	"os"
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
// Catppuccin palette v1.8.0: https://github.com/catppuccin/palette
// Keep palette values centralized and mapped to semantic TUI roles.
type themeColors struct {
	green, yellow, red, gray, blue, cyan, purple, indigo, orange, diskYellow string
	fg, muted, accent, header, surface, cursor, marked, base                 string
}

var catppuccinThemes = map[string]themeColors{
	"latte":     {"#40a02b", "#df8e1d", "#d20f39", "#7c7f93", "#1e66f5", "#179299", "#8839ef", "#7287fd", "#fe640b", "#df8e1d", "#4c4f69", "#5c5f77", "#11111b", "#bcc0cc", "#ccd0da", "#bcc0cc", "#e6e9ef", "#eff1f5"},
	"frappe":    {"#a6d189", "#e5c890", "#e78284", "#949cbb", "#8caaee", "#81c8be", "#ca9ee6", "#babbf1", "#ef9f76", "#e5c890", "#c6d0f5", "#b5bfe2", "#11111b", "#51576d", "#414559", "#292c3c", "#51576d", "#303446"},
	"macchiato": {"#a6da95", "#eed49f", "#ed8796", "#939ab7", "#8aadf4", "#8bd5ca", "#c6a0f6", "#b7bdf8", "#f5a97f", "#eed49f", "#cad3f5", "#b8c0e0", "#11111b", "#494d64", "#363a4f", "#1e2030", "#494d64", "#24273a"},
	"mocha":     {"#a6e3a1", "#f9e2af", "#f38ba8", "#9399b2", "#89b4fa", "#94e2d5", "#cba6f7", "#b4befe", "#fab387", "#f9e2af", "#cdd6f4", "#bac2de", "#11111b", "#45475a", "#313244", "#585b70", "#45475a", "#1e1e2e"},
}

var colorBase lipgloss.TerminalColor
var hasCustomTheme bool
var activeTheme string

func adaptive(hex string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: hex, Dark: hex}
}

func applyThemeColors(p themeColors) {
	colorGreen = adaptive(p.green)
	colorYellow = adaptive(p.yellow)
	colorRed = adaptive(p.red)
	colorGray = adaptive(p.gray)
	colorBlue = adaptive(p.blue)
	colorCyan = adaptive(p.cyan)
	colorPurple = adaptive(p.purple)
	colorIndigo = adaptive(p.indigo)
	colorOrange = adaptive(p.orange)
	colorDiskYellow = adaptive(p.diskYellow)
	colorFg = adaptive(p.fg)
	colorFgMuted = adaptive(p.muted)
	colorOnAccent = adaptive(p.accent)
	colorHeaderBg = adaptive(p.header)
	colorSurface = adaptive(p.surface)
	colorCursorBg = adaptive(p.cursor)
	colorMarkedBg = adaptive(p.marked)
	colorBase = lipgloss.Color(p.base)
}

// statusCardForeground uses a light foreground on Latte's red status card,
// whose dark-on-red contrast falls below WCAG AA for ordinary text.
func statusCardForeground(status string) lipgloss.TerminalColor {
	if activeTheme == "latte" && status == "red" {
		return lipgloss.Color("#ffffff")
	}
	return colorOnAccent
}

// SetTheme configures all TUI colors before Bubble Tea starts.
// This method updates cached package-level styles as well as per-cell colors.
func SetTheme(theme string) error {
	switch theme {
	case "latte", "frappe", "macchiato", "mocha":
		p := catppuccinThemes[theme]
		lipgloss.SetHasDarkBackground(theme != "latte")
		applyThemeColors(p)
		hasCustomTheme = true
	case "auto", "":
		// Retain the original adaptive palette for explicit auto mode.
		lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stdout))
		applyThemeColorsAuto()
		hasCustomTheme = false
	case "light", "dark":
		lipgloss.SetHasDarkBackground(theme == "dark")
		applyThemeColorsAuto()
		hasCustomTheme = false
	default:
		return fmt.Errorf("invalid theme %q: must be mocha, macchiato, frappe, latte, auto, light, or dark", theme)
	}
	activeTheme = theme
	refreshStyles()
	return nil
}

func applyThemeColorsAuto() {
	colorGreen = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#10b981"}
	colorYellow = lipgloss.AdaptiveColor{Light: "#b45309", Dark: "#f59e0b"}
	colorRed = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#ef4444"}
	colorGray = lipgloss.AdaptiveColor{Light: "#64748b", Dark: "#6b7280"}
	colorBlue = lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#3b82f6"}
	colorCyan = lipgloss.AdaptiveColor{Light: "#0e7490", Dark: "#06b6d4"}
	colorPurple = lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#8b5cf6"}
	colorIndigo = lipgloss.AdaptiveColor{Light: "#4f46e5", Dark: "#6366f1"}
	colorOrange = lipgloss.AdaptiveColor{Light: "#c2410c", Dark: "#f97316"}
	colorDiskYellow = lipgloss.AdaptiveColor{Light: "#a16207", Dark: "#eab308"}
	colorFg = lipgloss.AdaptiveColor{Light: "#0f172a", Dark: "#f8fafc"}
	colorFgMuted = lipgloss.AdaptiveColor{Light: "#334155", Dark: "#cbd5e1"}
	colorOnAccent = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#1e293b"}
	colorHeaderBg = lipgloss.AdaptiveColor{Light: "#e2e8f0", Dark: "#1e293b"}
	colorSurface = lipgloss.AdaptiveColor{Light: "#f1f5f9", Dark: "#0f172a"}
	colorCursorBg = lipgloss.AdaptiveColor{Light: "#bfdbfe", Dark: "#1e3a5f"}
	colorMarkedBg = lipgloss.AdaptiveColor{Light: "#c7d2fe", Dark: "#3730a3"}
	colorBase = nil
}

func refreshStyles() {
	StyleStatusGreen = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	StyleStatusYellow = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	StyleStatusRed = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	StyleStatusUnknown = lipgloss.NewStyle().Foreground(colorGray)
	StyleHeader = lipgloss.NewStyle().Background(colorHeaderBg).Foreground(colorFg).Padding(0, 1)
	StyleOverviewCard = lipgloss.NewStyle().Background(colorSurface).Foreground(colorFg).Padding(0, 1).Margin(0).Align(lipgloss.Center)
	StyleTableHeader = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(colorGray)
	StyleTableRow = lipgloss.NewStyle().Foreground(colorFg)
	StyleTableRowAlt = lipgloss.NewStyle().Foreground(colorFgMuted)
	StyleError = lipgloss.NewStyle().Foreground(colorRed).Bold(true)
	StyleDim = lipgloss.NewStyle().Foreground(colorGray)
	StyleGreen = lipgloss.NewStyle().Foreground(colorGreen)
	StyleYellow = lipgloss.NewStyle().Foreground(colorYellow)
	StyleOrange = lipgloss.NewStyle().Foreground(colorOrange)
	StyleBlue = lipgloss.NewStyle().Foreground(colorBlue)
	StyleCyan = lipgloss.NewStyle().Foreground(colorCyan)
	StylePurple = lipgloss.NewStyle().Foreground(colorPurple)
	StyleRed = lipgloss.NewStyle().Foreground(colorRed)
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
