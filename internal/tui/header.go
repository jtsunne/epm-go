package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// sanitize removes ANSI escape sequences and ASCII control characters from a
// string before rendering it in the terminal. This prevents a malicious or
// misbehaving server from injecting terminal control codes via cluster names
// or error messages.
//
// Handled escape sequence types:
//   - CSI (ESC [): terminated by a final byte in 0x40–0x7E
//   - String sequences (OSC ESC], DCS ESC P, PM ESC^, APC ESC_, SOS ESCX):
//     terminated by BEL (0x07) or ST (ESC \)
//   - Single-char escapes (all other ESC + one byte)
func sanitize(s string) string {
	var out strings.Builder
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		r := runes[i]
		if r != '\x1b' {
			// Printable: pass through. Control chars: skip.
			if r >= 0x20 && r != 0x7f && !(r >= 0x80 && r <= 0x9F) {
				out.WriteRune(r)
			}
			i++
			continue
		}
		// ESC: peek at the next rune to determine the sequence type.
		if i+1 >= len(runes) {
			i++ // lone ESC at end of string; skip
			continue
		}
		next := runes[i+1]
		switch next {
		case '[':
			// CSI sequence: ESC [ <intermediates> <final>
			// Final byte is in range 0x40–0x7E.
			i += 2 // skip ESC [
			for i < len(runes) && !(runes[i] >= 0x40 && runes[i] <= 0x7E) {
				i++
			}
			if i < len(runes) {
				i++ // skip final byte
			}
		case ']', 'P', '^', '_', 'X':
			// String-body sequences: OSC (]), DCS (P), PM (^), APC (_), SOS (X).
			// Terminated by BEL (0x07) or ST (ESC \).
			i += 2 // skip ESC and introducer
			for i < len(runes) {
				if runes[i] == '\x07' {
					i++ // skip BEL
					break
				}
				if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '\\' {
					i += 2 // skip ESC \
					break
				}
				i++
			}
		default:
			// Single-char escape: ESC followed by one byte.
			i += 2
		}
	}
	return out.String()
}

// classifyError returns a short, human-readable description of the connection
// error. Falls back to a truncated raw error string for unrecognised errors.
func classifyError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"):
		return "Connection refused"
	case strings.Contains(msg, "401") || strings.Contains(msg, "unauthorized"):
		return "Authentication failed (401)"
	case strings.Contains(msg, "403") || strings.Contains(msg, "forbidden"):
		return "Authentication failed (403)"
	case strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "timeout"):
		return "Timeout"
	case strings.Contains(msg, "certificate") || strings.Contains(msg, "tls") || strings.Contains(msg, "x509"):
		return "TLS error"
	default:
		raw := sanitize(err.Error())
		if len([]rune(raw)) > 40 {
			return string([]rune(raw)[:40]) + "..."
		}
		return raw
	}
}

// isTLSError reports whether err looks like a TLS/certificate error, in which
// case the UI should show a hint about the --insecure flag.
func isTLSError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "certificate") ||
		strings.Contains(msg, "tls") ||
		strings.Contains(msg, "x509")
}

// retryCountdown returns the right-side text shown in the header when
// disconnected: a live countdown to the next automatic retry attempt.
func retryCountdown(nextRetryAt time.Time) string {
	if nextRetryAt.IsZero() {
		return "Press r to retry"
	}
	remaining := time.Until(nextRetryAt)
	if remaining <= 0 {
		return "Retrying..."
	}
	secs := int(remaining.Seconds() + 0.5)
	if secs < 1 {
		secs = 1
	}
	return fmt.Sprintf("Retrying in %ds... (r: retry now)", secs)
}

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
			errLabel := classifyError(app.lastError)
			if isTLSError(app.lastError) {
				errLabel += "  (Try --insecure)"
			}
			center = StyleError.Render("● DISCONNECTED  " + errLabel)
			right = StyleError.Render(retryCountdown(app.nextRetryAt))
		} else {
			right = StyleDim.Render("Connecting...")
		}
	} else {
		// Have at least one snapshot — show cluster info.
		clusterName := sanitize(app.current.Health.ClusterName)
		if clusterName == "" && app.client != nil {
			clusterName = app.client.BaseURL()
		}
		left = clusterName

		if app.connState == stateDisconnected {
			// Lost connection after a successful fetch.
			errDisplay := "● DISCONNECTED"
			if app.lastError != nil {
				errLabel := classifyError(app.lastError)
				if isTLSError(app.lastError) {
					errLabel += "  (Try --insecure)"
				}
				errDisplay += "  " + errLabel
			}
			center = StyleError.Render(errDisplay)
			right = StyleError.Render(retryCountdown(app.nextRetryAt))
		} else {
			// Normal connected state.
			status := strings.ToUpper(sanitize(app.current.Health.Status))
			if status == "" {
				status = "UNKNOWN"
			}
			center = StatusStyle(app.current.Health.Status).Render("● " + status)

			lastStr := app.lastUpdated.Format("15:04:05")
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

	return StyleHeader.Width(width - 2).Render(row)
}

// formatDuration formats a poll interval as a compact string, e.g. "10s", "1m", or "1m30s".
func formatDuration(d time.Duration) string {
	if d >= time.Minute {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
