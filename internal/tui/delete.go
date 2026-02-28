package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jtsunne/epm-go/internal/client"
)

// deleteCmd issues a DELETE request for the given index names in a goroutine
// and returns a DeleteResultMsg with the outcome.
func deleteCmd(c client.ESClient, names []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := c.DeleteIndex(ctx, names)
		return DeleteResultMsg{Names: names, Err: err}
	}
}

// renderDeleteConfirm renders a full-screen confirmation dialog listing the
// indices pending deletion. The caller (View) renders the cluster header above
// and footer below; renderDeleteConfirm accounts for those heights.
func renderDeleteConfirm(app *App) string {
	width := app.width
	if width <= 0 {
		width = 80
	}
	height := app.height
	if height <= 0 {
		height = 24
	}

	// Title bar: styled like the cluster header.
	titleText := "Delete Index Confirmation"
	hintText := StyleDim.Render("[y: confirm  n/esc: cancel]")
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

	headerH := renderedHeight(renderHeader(app))
	footerH := renderedHeight(renderFooter(app))
	availH := height - headerH - titleH - footerH
	if availH < 1 {
		availH = 1
	}

	// Fixed header and footer lines that must always be visible.
	headerLines := []string{
		"",
		"  " + StyleRed.Bold(true).Render("WARNING: This action cannot be undone."),
		"",
		fmt.Sprintf("  The following %d index(es) will be permanently deleted:", len(app.pendingDeleteNames)),
		"",
	}
	footerLines := []string{
		"",
		"  " + StyleYellow.Render("Press y to confirm, n or esc to cancel."),
	}

	// Build the name list respecting available height.
	// footerLines (confirmation prompt) takes priority: trim nameLines first,
	// then headerLines from the bottom if needed. footerLines are never trimmed.
	nameLines := make([]string, 0, len(app.pendingDeleteNames))
	for _, name := range app.pendingDeleteNames {
		nameLines = append(nameLines, "    • "+sanitize(name))
	}

	fLen := len(footerLines)
	hLen := len(headerLines)

	// Space available for index names after reserving header and footer slots.
	nameSlots := availH - hLen - fLen
	if nameSlots < 0 {
		nameSlots = 0
	}

	// Truncate nameLines to fit nameSlots with an overflow indicator.
	displayNames := nameLines
	if len(nameLines) > nameSlots {
		switch {
		case nameSlots == 0:
			displayNames = nil
		case nameSlots == 1:
			displayNames = []string{fmt.Sprintf("    ...%d indices total", len(nameLines))}
		default:
			visible := nameSlots - 1
			hidden := len(nameLines) - visible
			dn := make([]string, visible+1)
			copy(dn, nameLines[:visible])
			dn[visible] = fmt.Sprintf("    ...and %d more", hidden)
			displayNames = dn
		}
	}

	// If header + footer alone exceed availH, trim header from the bottom
	// to protect footerLines (they contain the confirmation prompt).
	// If availH is smaller than fLen itself, trim footerLines from the top,
	// keeping only the last availH lines (the actual confirmation prompt).
	displayHeader := headerLines
	displayFooter := footerLines
	if hLen+fLen > availH {
		keep := availH - fLen
		if keep < 0 {
			keep = 0
			// availH < fLen: show only the last availH lines of the footer.
			trimFrom := fLen - availH
			displayFooter = footerLines[trimFrom:]
		}
		displayHeader = headerLines[:keep]
	}

	lines := make([]string, 0, availH)
	lines = append(lines, displayHeader...)
	lines = append(lines, displayNames...)
	lines = append(lines, displayFooter...)

	// Pad content area to availH.
	for len(lines) < availH {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return titleBar + "\n" + content
}
