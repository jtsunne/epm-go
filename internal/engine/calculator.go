package engine

// Sanity bounds ported from performanceTracker.ts lines 96-100.
const (
	minTimeDiffSeconds = 1.0
	maxRatePerSec      = 50_000_000.0
	maxLatencyMs       = 300_000.0
)

// clampRate returns 0 if r exceeds maxRatePerSec (counter wrap / bad data),
// otherwise returns r unchanged.
func clampRate(r float64) float64 {
	if r > maxRatePerSec {
		return 0
	}
	return r
}

// clampLatency caps l at maxLatencyMs.
func clampLatency(l float64) float64 {
	if l > maxLatencyMs {
		return maxLatencyMs
	}
	return l
}

// safeDivide returns a/b, or 0 when b is zero.
func safeDivide(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

// maxFloat64 returns the larger of a and b.
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
