package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderHeader renders the top header bar with cluster name, status, and timing info.
//
// Layout:
//   left:   cluster name (or "Connecting to <URL>..." on first connect)
//   center: colored "● STATUS" indicator (or "● DISCONNECTED  <error>" when offline)
//   right:  "Last: HH:MM:SS  Poll: Ns" (or "Press r to retry" when offline)
func renderHeader(app *App) string {
	width := app.width
	if width <= 0 {
		width = 80
	}

	var left, center, right string

	if app.current == nil {
		// No successful snapshot yet — initial connecting state.
		baseURL := ""
		if app.client != nil {
			baseURL = app.client.BaseURL()
		}
		left = "Connecting to " + baseURL + "..."

		if app.connState == stateDisconnected && app.lastError != nil {
			errMsg := app.lastError.Error()
			if len(errMsg) > 40 {
				errMsg = errMsg[:40] + "..."
			}
			center = StyleError.Render("● DISCONNECTED  " + errMsg)
			right = StyleError.Render("Press r to retry")
		}
	} else {
		// Have at least one snapshot — show cluster info.
		clusterName := app.current.Health.ClusterName
		if clusterName == "" && app.client != nil {
			clusterName = app.client.BaseURL()
		}
		left = clusterName

		if app.connState == stateDisconnected {
			// Lost connection after a successful fetch.
			errDisplay := "● DISCONNECTED"
			if app.lastError != nil {
				errMsg := app.lastError.Error()
				if len(errMsg) > 40 {
					errMsg = errMsg[:40] + "..."
				}
				errDisplay += "  " + errMsg
			}
			center = StyleError.Render(errDisplay)
			right = StyleError.Render("Press r to retry")
		} else {
			// Normal connected state.
			status := strings.ToUpper(app.current.Health.Status)
			if status == "" {
				status = "UNKNOWN"
			}
			center = StatusStyle(app.current.Health.Status).Render("● " + status)

			lastStr := "Connecting..."
			if !app.lastUpdated.IsZero() {
				lastStr = app.lastUpdated.Format("15:04:05")
			}
			right = StyleDim.Render(fmt.Sprintf("Last: %s  Poll: %s", lastStr, formatDuration(app.pollInterval)))
		}
	}

	// Build row: left + padding + center + padding + right, filling innerWidth.
	// StyleHeader has Padding(0, 1) so inner content width = total width - 2.
	innerWidth := width - 2
	leftVW := lipgloss.Width(left)
	centerVW := lipgloss.Width(center)
	rightVW := lipgloss.Width(right)

	spacing := innerWidth - leftVW - centerVW - rightVW
	if spacing < 0 {
		spacing = 0
	}
	leftSpacing := spacing / 2
	rightSpacing := spacing - leftSpacing

	row := left +
		strings.Repeat(" ", leftSpacing) +
		center +
		strings.Repeat(" ", rightSpacing) +
		right

	return StyleHeader.Width(width).Render(row)
}

// formatDuration formats a poll interval as a compact string, e.g. "10s" or "2m".
func formatDuration(d time.Duration) string {
	if d >= time.Minute {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
