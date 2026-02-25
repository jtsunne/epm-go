package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/jtsunne/epm-go/internal/model"
)

// categoryLabel returns the display name for a recommendation category.
func categoryLabel(cat model.RecommendationCategory) string {
	switch cat {
	case model.CategoryResourcePressure:
		return "Resource Pressure"
	case model.CategoryShardHealth:
		return "Shard Health"
	case model.CategoryIndexConfig:
		return "Index Configuration"
	case model.CategoryHotspot:
		return "Hotspot"
	default:
		return "Other"
	}
}

// severityBadge returns a colored, fixed-width badge for the given severity.
func severityBadge(sev model.RecommendationSeverity) string {
	switch sev {
	case model.SeverityCritical:
		return StyleRed.Bold(true).Render("[CRITICAL]")
	case model.SeverityWarning:
		return StyleYellow.Bold(true).Render("[WARN]    ")
	default:
		return StyleGreen.Bold(true).Render("[OK]      ")
	}
}

// wrapText wraps text at maxWidth rune-columns, breaking at word boundaries.
// Returns the original string unchanged when it fits within maxWidth.
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 || utf8.RuneCountInString(text) <= maxWidth {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var lines []string
	var current strings.Builder
	var currentLen int // rune count of current line
	for _, word := range words {
		wordLen := utf8.RuneCountInString(word)
		if currentLen == 0 {
			current.WriteString(word)
			currentLen = wordLen
		} else if currentLen+1+wordLen <= maxWidth {
			current.WriteByte(' ')
			current.WriteString(word)
			currentLen += 1 + wordLen
		} else {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
			currentLen = wordLen
		}
	}
	if currentLen > 0 {
		lines = append(lines, current.String())
	}
	return strings.Join(lines, "\n")
}

// buildAnalyticsLines returns the full list of rendered content lines for the
// analytics view. Extracted so the same logic can be used both during rendering
// and when computing the maximum scroll offset in Update().
func buildAnalyticsLines(recs []model.Recommendation, width int) []string {
	var lines []string
	if len(recs) == 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+StyleGreen.Bold(true).Render("No issues found — cluster looks healthy"))
		lines = append(lines, "")
	} else {
		categories := []model.RecommendationCategory{
			model.CategoryResourcePressure,
			model.CategoryShardHealth,
			model.CategoryIndexConfig,
			model.CategoryHotspot,
		}
		for _, cat := range categories {
			var catRecs []model.Recommendation
			for _, r := range recs {
				if r.Category == cat {
					catRecs = append(catRecs, r)
				}
			}
			if len(catRecs) == 0 {
				continue
			}
			catHeader := StyleDim.Bold(true).Underline(true).Render(categoryLabel(cat))
			lines = append(lines, "")
			lines = append(lines, "  "+catHeader)
			for _, r := range catRecs {
				badge := severityBadge(r.Severity)
				lines = append(lines, fmt.Sprintf("  %s %s", badge, r.Title))
				if r.Detail != "" {
					wrapped := wrapText(r.Detail, width-6)
					for _, dline := range strings.Split(wrapped, "\n") {
						lines = append(lines, "    "+dline)
					}
				}
			}
		}
	}
	return lines
}

// renderAnalyticsTitle renders the title bar for the analytics screen and
// returns the styled string. Extracted so that both renderAnalytics and
// analyticsMaxOffset measure the same rendered height instead of assuming a
// constant of 1 line (which breaks on narrow terminals where the title wraps).
func renderAnalyticsTitle(width int) string {
	const titleText = "Analytics — Cluster Recommendations"
	hintText := StyleDim.Render("[a/esc: back]")
	hintVW := lipgloss.Width(hintText)
	titleVW := lipgloss.Width(titleText)
	innerWidth := width - 2 // StyleHeader has Padding(0,1) -> 1 char per side
	gap := innerWidth - titleVW - hintVW
	if gap < 1 {
		gap = 1
	}
	titleRow := titleText + strings.Repeat(" ", gap) + hintText
	return StyleHeader.Width(width).MaxWidth(width).Render(titleRow)
}

// analyticsMaxOffset returns the maximum valid analyticsScrollOffset for the
// current app state. Called from Update() to clamp the stored offset after a
// CursorDown key event, preventing overscroll debt where the stored offset
// exceeds the real content bound and subsequent CursorUp presses appear
// non-responsive because the display stays clamped until the debt is paid down.
func analyticsMaxOffset(app *App) int {
	width := app.width
	if width <= 0 {
		width = 80
	}
	height := app.height
	if height <= 0 {
		height = 24
	}
	headerH := renderedHeight(renderHeader(app))
	titleH := renderedHeight(renderAnalyticsTitle(width))
	footerH := renderedHeight(renderFooter(app))
	availH := height - headerH - titleH - footerH
	if availH < 1 {
		availH = 1
	}
	lines := buildAnalyticsLines(app.recommendations, width)
	overflows := len(lines) > availH
	contentH := availH
	if overflows && contentH > 1 {
		contentH--
	}
	max := len(lines) - contentH
	if max < 0 {
		max = 0
	}
	return max
}

// renderAnalytics renders the analytics title bar followed by the scrollable
// recommendations list. The caller (View) renders the cluster header above and
// footer below; renderAnalytics accounts for those heights when computing the
// available content height so the full layout exactly fills the terminal.
func renderAnalytics(app *App) string {
	width := app.width
	if width <= 0 {
		width = 80
	}
	height := app.height
	if height <= 0 {
		height = 24
	}

	// Title bar: left title + right hint, styled like the cluster header.
	titleBar := renderAnalyticsTitle(width)
	titleH := lipgloss.Height(titleBar)

	// Available lines for scrollable content: total height minus the sections
	// rendered outside this function (cluster header, analytics title, footer).
	headerH := renderedHeight(renderHeader(app))
	footerH := renderedHeight(renderFooter(app))
	availH := height - headerH - titleH - footerH
	if availH < 1 {
		availH = 1
	}

	// Build the full list of content lines.
	lines := buildAnalyticsLines(app.recommendations, width)

	// When content overflows, reserve the last line for a scroll hint.
	overflows := len(lines) > availH
	contentH := availH
	if overflows && contentH > 1 {
		contentH--
	}

	// Clamp scroll offset to valid range (read-only; model state is not mutated in View).
	maxOffset := len(lines) - contentH
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := app.analyticsScrollOffset
	if offset > maxOffset {
		offset = maxOffset
	}

	// Slice visible content lines.
	end := offset + contentH
	if end > len(lines) {
		end = len(lines)
	}
	var visibleLines []string
	if offset < len(lines) {
		visibleLines = append(visibleLines, lines[offset:end]...)
	}

	// Pad content area to contentH with empty lines.
	for len(visibleLines) < contentH {
		visibleLines = append(visibleLines, "")
	}

	// Append scroll hint as its own line (does not overwrite content).
	if overflows {
		var hint string
		if offset == 0 {
			hint = StyleDim.Render("  ↓ scroll for more")
		} else if offset >= maxOffset {
			hint = StyleDim.Render("  ↑ scroll up")
		} else {
			hint = StyleDim.Render("  ↑↓ scroll")
		}
		visibleLines = append(visibleLines, hint)
	}

	content := strings.Join(visibleLines, "\n")
	return titleBar + "\n" + content
}
