package tui

import (
	"fmt"
	"strings"

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

// wrapText wraps text at maxWidth characters, breaking at word boundaries.
// Returns the original string unchanged when it fits within maxWidth.
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 || len(text) <= maxWidth {
		return text
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var lines []string
	var current strings.Builder
	for _, word := range words {
		if current.Len() == 0 {
			current.WriteString(word)
		} else if current.Len()+1+len(word) <= maxWidth {
			current.WriteByte(' ')
			current.WriteString(word)
		} else {
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return strings.Join(lines, "\n")
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
	titleBar := StyleHeader.Width(width).MaxWidth(width).Render(titleRow)
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
	var lines []string
	recs := app.recommendations
	if len(recs) == 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+StyleGreen.Bold(true).Render("No issues found — cluster looks healthy"))
		lines = append(lines, "")
	} else {
		// Render categories in a fixed display order.
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

	// Clamp scroll offset to valid range.
	maxOffset := len(lines) - availH
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := app.analyticsScrollOffset
	if offset > maxOffset {
		offset = maxOffset
	}

	// Slice visible lines.
	end := offset + availH
	if end > len(lines) {
		end = len(lines)
	}
	var visibleLines []string
	if offset < len(lines) {
		visibleLines = append(visibleLines, lines[offset:end]...)
	}

	// Pad to fill availH with empty lines.
	for len(visibleLines) < availH {
		visibleLines = append(visibleLines, "")
	}

	// Show a scroll hint on the last visible line when content overflows.
	if len(lines) > availH && len(visibleLines) > 0 {
		var hint string
		if offset == 0 {
			hint = StyleDim.Render("  ↓ scroll for more")
		} else if offset >= maxOffset {
			hint = StyleDim.Render("  ↑ scroll up")
		} else {
			hint = StyleDim.Render("  ↑↓ scroll")
		}
		visibleLines[len(visibleLines)-1] = hint
	}

	content := strings.Join(visibleLines, "\n")
	return titleBar + "\n" + content
}
