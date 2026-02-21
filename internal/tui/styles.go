package tui

import "github.com/charmbracelet/lipgloss"

// Color constants — ES Performance Monitor palette.
var (
	colorGreen  = lipgloss.Color("#10b981")
	colorYellow = lipgloss.Color("#f59e0b")
	colorRed    = lipgloss.Color("#ef4444")
	colorGray   = lipgloss.Color("#6b7280")
	colorBlue   = lipgloss.Color("#3b82f6")
	colorCyan   = lipgloss.Color("#06b6d4")
	colorPurple = lipgloss.Color("#8b5cf6")
	colorIndigo = lipgloss.Color("#6366f1")
	colorOrange = lipgloss.Color("#f97316")
	colorWhite  = lipgloss.Color("#f8fafc")
	colorDark   = lipgloss.Color("#1e293b")
	colorAlt    = lipgloss.Color("#0f172a")
)

// Status styles — bold foreground, used for cluster health indicator.
var (
	StyleStatusGreen   = lipgloss.NewStyle().Bold(true).Foreground(colorGreen)
	StyleStatusYellow  = lipgloss.NewStyle().Bold(true).Foreground(colorYellow)
	StyleStatusRed     = lipgloss.NewStyle().Bold(true).Foreground(colorRed)
	StyleStatusUnknown = lipgloss.NewStyle().Foreground(colorGray)
)

// StyleHeader — full-width dark header bar.
var StyleHeader = lipgloss.NewStyle().
	Background(colorDark).
	Foreground(colorWhite).
	Padding(0, 1)

// StyleOverviewCard — bordered card for the 7-stat overview bar.
var StyleOverviewCard = lipgloss.NewStyle().
	Background(colorAlt).
	Foreground(colorWhite).
	Padding(0, 1).
	Margin(0).
	Align(lipgloss.Center)

// StyleMetricCard — card for the 4 metric sparkline panels (Phase 4).
var StyleMetricCard = lipgloss.NewStyle().
	Background(colorAlt).
	Foreground(colorWhite).
	Padding(0, 1).
	Margin(0)

// Table styles.
var (
	StyleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Underline(true).
				Foreground(colorGray)

	StyleTableRow = lipgloss.NewStyle().
			Foreground(colorWhite)

	StyleTableRowAlt = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cbd5e1"))
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
// Accepts "green", "yellow", "red" (case-insensitive via lowercase comparison).
func StatusStyle(status string) lipgloss.Style {
	switch status {
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
