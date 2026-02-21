package model

import "time"

const defaultSparklineCap = 60

// SparklinePoint is a single timestamped data point stored in the ring buffer.
type SparklinePoint struct {
	Timestamp     time.Time
	IndexingRate  float64
	SearchRate    float64
	IndexLatency  float64
	SearchLatency float64
}

// SparklineHistory is a fixed-size ring buffer of SparklinePoints.
// When the buffer is full, new pushes overwrite the oldest entry.
type SparklineHistory struct {
	buf  []SparklinePoint
	head int // index of the next write position
	size int // number of valid entries
}

// NewSparklineHistory creates a SparklineHistory with the given capacity.
// If cap <= 0, the defaultSparklineCap (60) is used.
func NewSparklineHistory(capacity int) *SparklineHistory {
	if capacity <= 0 {
		capacity = defaultSparklineCap
	}
	return &SparklineHistory{
		buf: make([]SparklinePoint, capacity),
	}
}

// Push appends a new point to the history, overwriting the oldest if full.
func (h *SparklineHistory) Push(p SparklinePoint) {
	h.buf[h.head] = p
	h.head = (h.head + 1) % len(h.buf)
	if h.size < len(h.buf) {
		h.size++
	}
}

// Len returns the number of valid entries in the history.
func (h *SparklineHistory) Len() int {
	return h.size
}

// Clear resets the history to empty.
func (h *SparklineHistory) Clear() {
	h.head = 0
	h.size = 0
}

// Values returns a slice of float64 for the named field in chronological order
// (oldest first). Valid field names: "indexingRate", "searchRate",
// "indexLatency", "searchLatency".
func (h *SparklineHistory) Values(field string) []float64 {
	out := make([]float64, h.size)
	// oldest entry sits at (head - size + cap) % cap
	start := (h.head - h.size + len(h.buf)) % len(h.buf)
	for i := 0; i < h.size; i++ {
		p := h.buf[(start+i)%len(h.buf)]
		switch field {
		case "indexingRate":
			out[i] = p.IndexingRate
		case "searchRate":
			out[i] = p.SearchRate
		case "indexLatency":
			out[i] = p.IndexLatency
		case "searchLatency":
			out[i] = p.SearchLatency
		}
	}
	return out
}
