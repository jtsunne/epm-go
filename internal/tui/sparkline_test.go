package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// testColor is a neutral color used for sparkline tests.
var testColor = lipgloss.Color("#ffffff")

// plainText strips ANSI escape sequences from a string so tests can assert
// on visible character content without caring about color codes.
func plainText(s string) string {
	out := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip ESC[ ... m sequence.
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip 'm'
			continue
		}
		out = append(out, s[i])
		i++
	}
	return string(out)
}

func TestRenderSparkline_Empty(t *testing.T) {
	result := plainText(RenderSparkline(nil, 10, testColor))
	if result != strings.Repeat(" ", 10) {
		t.Errorf("expected 10 spaces, got %q", result)
	}
}

func TestRenderSparkline_EmptySlice(t *testing.T) {
	result := plainText(RenderSparkline([]float64{}, 10, testColor))
	if result != strings.Repeat(" ", 10) {
		t.Errorf("expected 10 spaces, got %q", result)
	}
}

func TestRenderSparkline_AllZeros(t *testing.T) {
	values := []float64{0, 0, 0, 0, 0}
	result := plainText(RenderSparkline(values, 5, testColor))
	for i, ch := range []rune(result) {
		if ch != '▁' {
			t.Errorf("index %d: expected '▁', got %q", i, ch)
		}
	}
}

func TestRenderSparkline_Ascending(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8}
	result := []rune(plainText(RenderSparkline(values, 8, testColor)))

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
	result := []rune(plainText(RenderSparkline(values, 10, testColor)))

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
	result := []rune(plainText(RenderSparkline([]float64{42}, 5, testColor)))

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
