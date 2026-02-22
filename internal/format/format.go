package format

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatBytes formats a byte count into a human-readable string with 1 decimal place.
// Thresholds: <1KB → B, <1MB → KB, <1GB → MB, <1TB → GB, else TB.
func FormatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)
	switch {
	case bytes < kb:
		return fmt.Sprintf("%d B", bytes)
	case bytes < mb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/kb)
	case bytes < gb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/mb)
	case bytes < tb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/gb)
	default:
		return fmt.Sprintf("%.1f TB", float64(bytes)/tb)
	}
}

// FormatRate formats an ops/sec rate with comma-separated thousands and one decimal place.
// Example: 1204.3 → "1,204.3 /s", 0 → "0 /s".
// Negative values (sentinel MetricNotAvailable) return "---".
func FormatRate(opsPerSec float64) string {
	if opsPerSec < 0 {
		return "---"
	}
	if opsPerSec == 0 {
		return "0 /s"
	}
	return formatCommaFloat(opsPerSec) + " /s"
}

// FormatLatency formats a latency value in milliseconds.
// Values >= 1000 ms are shown as seconds with 2 decimal places.
// Values < 1000 ms are shown as ms with 2 decimal places.
// Negative values (sentinel MetricNotAvailable) return "---".
func FormatLatency(ms float64) string {
	if ms < 0 {
		return "---"
	}
	if ms >= 1000 {
		return fmt.Sprintf("%.2f s", ms/1000)
	}
	return fmt.Sprintf("%.2f ms", ms)
}

// FormatNumber formats an integer with locale-style comma separators.
// Example: 12345678 → "12,345,678".
// Uses strconv.FormatInt directly to avoid abs64 overflow for math.MinInt64.
func FormatNumber(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		// s starts with "-"; strip it, insert commas, restore sign.
		return "-" + insertCommas(s[1:])
	}
	return insertCommas(s)
}

// FormatPercent formats a percentage with one decimal place.
// Example: 34.5 → "34.5%".
func FormatPercent(p float64) string {
	return fmt.Sprintf("%.1f%%", p)
}

// formatCommaFloat formats a float with comma-separated thousands and one decimal place.
func formatCommaFloat(f float64) string {
	// Format with one decimal place first
	formatted := fmt.Sprintf("%.1f", f)
	// Strip leading minus before inserting commas, then restore it
	sign := ""
	if len(formatted) > 0 && formatted[0] == '-' {
		sign = "-"
		formatted = formatted[1:]
	}
	// Split on decimal point
	parts := strings.SplitN(formatted, ".", 2)
	intPart := insertCommas(parts[0])
	if len(parts) == 2 {
		return sign + intPart + "." + parts[1]
	}
	return sign + intPart
}

// insertCommas inserts comma separators into a digit string every 3 digits from the right.
func insertCommas(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var buf strings.Builder
	lead := n % 3
	if lead > 0 {
		buf.WriteString(s[:lead])
	}
	for i := lead; i < n; i += 3 {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(s[i : i+3])
	}
	return buf.String()
}
