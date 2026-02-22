package tui

import "github.com/charmbracelet/lipgloss"

// severity represents the alert level for a metric value.
type severity int

const (
	severityNormal   severity = iota
	severityWarning           // yellow
	severityCritical          // red
)

// cpuSeverity returns Warning when CPU > 80%, Critical when > 90%.
func cpuSeverity(pct float64) severity {
	switch {
	case pct > 90:
		return severityCritical
	case pct > 80:
		return severityWarning
	default:
		return severityNormal
	}
}

// jvmSeverity returns Warning when JVM heap > 75%, Critical when > 85%.
func jvmSeverity(pct float64) severity {
	switch {
	case pct > 85:
		return severityCritical
	case pct > 75:
		return severityWarning
	default:
		return severityNormal
	}
}

// storageSeverity returns Warning when storage > 80%, Critical when > 90%.
func storageSeverity(pct float64) severity {
	switch {
	case pct > 90:
		return severityCritical
	case pct > 80:
		return severityWarning
	default:
		return severityNormal
	}
}

// searchLatSeverity returns Critical when search latency > 1000ms.
func searchLatSeverity(ms float64) severity {
	if ms > 1000 {
		return severityCritical
	}
	return severityNormal
}

// indexLatSeverity returns Warning when index latency > 500ms.
func indexLatSeverity(ms float64) severity {
	if ms > 500 {
		return severityWarning
	}
	return severityNormal
}

// severityToStyle maps a severity level to the appropriate lipgloss style.
func severityToStyle(s severity) lipgloss.Style {
	switch s {
	case severityWarning:
		return StyleYellow
	case severityCritical:
		return StyleRed
	default:
		return lipgloss.NewStyle()
	}
}
