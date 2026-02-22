package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// testColor is a neutral color used for sparkline tests.
var testColor = lipgloss.Color("#ffffff")

func TestRenderSparkline_Empty(t *testing.T) {
	result := stripANSI(RenderSparkline(nil, 10, testColor))
	if result != strings.Repeat(" ", 10) {
		t.Errorf("expected 10 spaces, got %q", result)
	}
}

func TestRenderSparkline_EmptySlice(t *testing.T) {
	result := stripANSI(RenderSparkline([]float64{}, 10, testColor))
	if result != strings.Repeat(" ", 10) {
		t.Errorf("expected 10 spaces, got %q", result)
	}
}

func TestRenderSparkline_AllZeros(t *testing.T) {
	values := []float64{0, 0, 0, 0, 0}
	result := stripANSI(RenderSparkline(values, 5, testColor))
	runes := []rune(result)
	if len(runes) != 5 {
		t.Fatalf("expected 5 runes, got %d: %q", len(runes), result)
	}
	for i, ch := range runes {
		if ch != '▁' {
			t.Errorf("index %d: expected '▁', got %q", i, ch)
		}
	}
}

func TestRenderSparkline_Ascending(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	result := []rune(stripANSI(RenderSparkline(values, 8, testColor)))

	if len(result) != 8 {
		t.Fatalf("expected 8 runes, got %d: %q", len(result), string(result))
	}

	// Characters should be non-decreasing left to right.
	for i := 1; i < len(result); i++ {
		if result[i] < result[i-1] {
			t.Errorf("index %d: expected non-decreasing, got %q < %q", i, result[i], result[i-1])
		}
	}

	// Last character should be '█' (max value maps to index 7).
	if result[7] != '█' {
		t.Errorf("last char: expected '█', got %q", result[7])
	}
}

func TestRenderSparkline_TruncatesLeft(t *testing.T) {
	// 20 values; width=10 → only the last 10 values are used.
	values := make([]float64, 20)
	for i := range values {
		values[i] = float64(i)
	}
	result := []rune(stripANSI(RenderSparkline(values, 10, testColor)))

	if len(result) != 10 {
		t.Fatalf("expected 10 runes, got %d", len(result))
	}

	// Last 10 values are 10..19; the last visible char should be '█'.
	if result[9] != '█' {
		t.Errorf("expected last char '█', got %q", result[9])
	}
}

func TestRenderSparkline_SingleValue(t *testing.T) {
	// 1 value, width=5 → 4 spaces + one '█'
	result := []rune(stripANSI(RenderSparkline([]float64{42}, 5, testColor)))

	if len(result) != 5 {
		t.Fatalf("expected 5 runes, got %d", len(result))
	}
	for i := 0; i < 4; i++ {
		if result[i] != ' ' {
			t.Errorf("index %d: expected space, got %q", i, result[i])
		}
	}
	if result[4] != '█' {
		t.Errorf("index 4: expected '█', got %q", result[4])
	}
}

func TestRenderSparkline_ZeroWidth(t *testing.T) {
	result := RenderSparkline([]float64{1, 2, 3}, 0, testColor)
	if result != "" {
		t.Errorf("expected empty string for width=0, got %q", result)
	}
}

func TestRenderSparkline_AllSentinel(t *testing.T) {
	// All MetricNotAvailable (-1.0) values: maxVal <= 0, so every bar renders at floor.
	values := []float64{-1, -1, -1, -1, -1}
	result := []rune(stripANSI(RenderSparkline(values, 5, testColor)))
	if len(result) != 5 {
		t.Fatalf("expected 5 runes, got %d: %q", len(result), string(result))
	}
	for i, ch := range result {
		if ch != '▁' {
			t.Errorf("index %d: expected '▁' (floor) for sentinel, got %q", i, ch)
		}
	}
}

func TestRenderSparkline_MixedSentinel(t *testing.T) {
	// Sentinel values mixed with real values: sentinel renders at floor level (clamped to 0).
	values := []float64{-1, 5, -1}
	result := []rune(stripANSI(RenderSparkline(values, 3, testColor)))
	if len(result) != 3 {
		t.Fatalf("expected 3 runes, got %d: %q", len(result), string(result))
	}
	// First and last are sentinel (-1), which produces negative idx clamped to 0 → '▁'
	if result[0] != '▁' {
		t.Errorf("index 0: expected '▁' for sentinel, got %q", result[0])
	}
	// Middle is the max value → '█'
	if result[1] != '█' {
		t.Errorf("index 1: expected '█' for max value, got %q", result[1])
	}
	if result[2] != '▁' {
		t.Errorf("index 2: expected '▁' for sentinel, got %q", result[2])
	}
}
