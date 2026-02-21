package format

import (
	"fmt"
	"math"
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
func FormatRate(opsPerSec float64) string {
	if opsPerSec == 0 {
		return "0 /s"
	}
	return formatCommaFloat(opsPerSec) + " /s"
}

// FormatLatency formats a latency value in milliseconds.
// Values >= 1000 ms are shown as seconds with 2 decimal places.
// Values < 1000 ms are shown as ms with 2 decimal places.
func FormatLatency(ms float64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.2f s", ms/1000)
	}
	return fmt.Sprintf("%.2f ms", ms)
}

// FormatNumber formats an integer with locale-style comma separators.
// Example: 12345678 → "12,345,678".
func FormatNumber(n int64) string {
	s := strconv.FormatInt(abs64(n), 10)
	result := insertCommas(s)
	if n < 0 {
		return "-" + result
	}
	return result
}

// FormatPercent formats a percentage with one decimal place.
// Example: 34.5 → "34.5%".
func FormatPercent(p float64) string {
	return fmt.Sprintf("%.1f%%", p)
}

// ParseHumanBytes parses a human-readable byte string (e.g. "20.4gb", "100mb", "1.5tb")
// into the equivalent number of bytes. Returns 0 on parse failure.
// Supported suffixes (case-insensitive): b, kb, mb, gb, tb.
func ParseHumanBytes(s string) int64 {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0
	}

	suffixes := []struct {
		suffix string
		mult   float64
	}{
		{"tb", 1024 * 1024 * 1024 * 1024},
		{"gb", 1024 * 1024 * 1024},
		{"mb", 1024 * 1024},
		{"kb", 1024},
		{"b", 1},
	}

	for _, entry := range suffixes {
		if strings.HasSuffix(s, entry.suffix) {
			numStr := strings.TrimSuffix(s, entry.suffix)
			numStr = strings.TrimSpace(numStr)
			val, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0
			}
			return int64(math.Round(val * entry.mult))
		}
	}
	// No suffix — try plain integer
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// formatCommaFloat formats a float with comma-separated thousands and one decimal place.
func formatCommaFloat(f float64) string {
	// Format with one decimal place first
	formatted := fmt.Sprintf("%.1f", f)
	// Split on decimal point
	parts := strings.SplitN(formatted, ".", 2)
	intPart := insertCommas(parts[0])
	if len(parts) == 2 {
		return intPart + "." + parts[1]
	}
	return intPart
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

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
