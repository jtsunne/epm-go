package tui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtsunne/epm-go/internal/model"
)

func TestCategoryLabel(t *testing.T) {
	cases := []struct {
		cat  model.RecommendationCategory
		want string
	}{
		{model.CategoryResourcePressure, "Resource Pressure"},
		{model.CategoryShardHealth, "Shard Health"},
		{model.CategoryIndexConfig, "Index Configuration"},
		{model.CategoryHotspot, "Hotspot"},
		{model.CategoryIndexLifecycle, "Index Lifecycle"},
		{model.RecommendationCategory(99), "Other"},
	}
	for _, tc := range cases {
		got := categoryLabel(tc.cat)
		assert.Equal(t, tc.want, got)
	}
}

func TestSeverityBadge_NotEmpty(t *testing.T) {
	cases := []model.RecommendationSeverity{
		model.SeverityNormal,
		model.SeverityWarning,
		model.SeverityCritical,
	}
	for _, sev := range cases {
		badge := severityBadge(sev)
		assert.NotEmpty(t, badge)
		stripped := stripANSI(badge)
		assert.NotEmpty(t, stripped)
	}
}

func TestSeverityBadge_ContainsExpectedText(t *testing.T) {
	assert.Contains(t, stripANSI(severityBadge(model.SeverityCritical)), "CRITICAL")
	assert.Contains(t, stripANSI(severityBadge(model.SeverityWarning)), "WARN")
	assert.Contains(t, stripANSI(severityBadge(model.SeverityNormal)), "OK")
}

func TestWrapText(t *testing.T) {
	cases := []struct {
		name      string
		text      string
		maxWidth  int
		wantLines int
	}{
		{"fits in one line", "hello world", 20, 1},
		{"exactly max", "hello", 5, 1},
		{"needs wrapping", "one two three four five", 12, 3},
		{"zero width returns as-is", "hello world", 0, 1},
		{"single long word", "superlongword", 5, 1}, // can't break within a word
		{"empty string", "", 10, 1},
		// Multi-byte UTF-8: "≤" is 3 bytes but 1 rune-column; wrapping must count runes.
		{"unicode multibyte fits", "data ≤30×", 10, 1},
		{"unicode multibyte wraps", "Index data is 25× total heap. Elastic recommends ≤30×.", 30, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := wrapText(tc.text, tc.maxWidth)
			lines := strings.Split(result, "\n")
			assert.Equal(t, tc.wantLines, len(lines), "got: %q", result)
		})
	}
}

func TestWrapText_LinesRespectMaxWidth(t *testing.T) {
	text := "Average cluster CPU at 92%. Critical load risks query timeouts. Add data nodes or reduce indexing throughput."
	maxWidth := 40
	result := wrapText(text, maxWidth)
	for i, line := range strings.Split(result, "\n") {
		assert.LessOrEqual(t, utf8.RuneCountInString(line), maxWidth, "line %d exceeds max width: %q", i, line)
	}
}

// makeAnalyticsApp builds a minimal App with a snapshot and fixed dimensions.
func makeAnalyticsApp() *App {
	app := NewApp(nil, 10*time.Second)
	app.width = 100
	app.height = 30
	app.current = &model.Snapshot{FetchedAt: time.Now()}
	app.current.Health.ClusterName = "test-cluster"
	app.current.Health.Status = "green"
	return app
}

func TestRenderAnalytics_EmptyRecommendations(t *testing.T) {
	app := makeAnalyticsApp()
	app.analyticsMode = true
	app.recommendations = []model.Recommendation{}

	result := renderAnalytics(app)
	require.NotEmpty(t, result)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "Analytics")
	assert.Contains(t, stripped, "No issues found")
	assert.Contains(t, stripped, "a/esc: back")
}

func TestRenderAnalytics_NilRecommendations(t *testing.T) {
	app := makeAnalyticsApp()
	app.analyticsMode = true
	app.recommendations = nil

	result := renderAnalytics(app)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "No issues found")
}

func TestRenderAnalytics_WithRecommendations(t *testing.T) {
	app := makeAnalyticsApp()
	app.analyticsMode = true
	app.recommendations = []model.Recommendation{
		{
			Severity: model.SeverityCritical,
			Category: model.CategoryResourcePressure,
			Title:    "High CPU usage",
			Detail:   "Average cluster CPU at 92%.",
		},
		{
			Severity: model.SeverityWarning,
			Category: model.CategoryShardHealth,
			Title:    "Uneven shard distribution",
			Detail:   "Some nodes have too many shards.",
		},
		{
			Severity: model.SeverityNormal,
			Category: model.CategoryIndexConfig,
			Title:    "Index retention OK",
			Detail:   "",
		},
	}

	result := renderAnalytics(app)
	stripped := stripANSI(result)

	assert.Contains(t, stripped, "Analytics")
	assert.Contains(t, stripped, "Resource Pressure")
	assert.Contains(t, stripped, "High CPU usage")
	assert.Contains(t, stripped, "92%")
	assert.Contains(t, stripped, "Shard Health")
	assert.Contains(t, stripped, "Uneven shard distribution")
	assert.Contains(t, stripped, "Index Configuration")
	assert.Contains(t, stripped, "Index retention OK")
	// Severity badges
	assert.Contains(t, stripped, "CRITICAL")
	assert.Contains(t, stripped, "WARN")
	assert.Contains(t, stripped, "OK")
}

func TestRenderAnalytics_CategoryOrder(t *testing.T) {
	app := makeAnalyticsApp()
	app.analyticsMode = true
	app.recommendations = []model.Recommendation{
		{Severity: model.SeverityWarning, Category: model.CategoryHotspot, Title: "Hotspot item"},
		{Severity: model.SeverityCritical, Category: model.CategoryResourcePressure, Title: "Resource item"},
	}

	result := renderAnalytics(app)
	stripped := stripANSI(result)

	idxResource := strings.Index(stripped, "Resource Pressure")
	idxHotspot := strings.Index(stripped, "Hotspot")
	assert.Less(t, idxResource, idxHotspot, "ResourcePressure should appear before Hotspot")
}

func TestRenderAnalytics_ScrollHintAppearsWhenOverflow(t *testing.T) {
	app := makeAnalyticsApp()
	app.height = 10 // very small terminal
	app.analyticsMode = true
	// Fill with enough recommendations to exceed height.
	for i := 0; i < 20; i++ {
		app.recommendations = append(app.recommendations, model.Recommendation{
			Severity: model.SeverityWarning,
			Category: model.CategoryShardHealth,
			Title:    "Recommendation item",
			Detail:   "Some detail text here.",
		})
	}

	result := renderAnalytics(app)
	stripped := stripANSI(result)
	// The scroll hint text "scroll" must appear in the rendered output.
	// We check for "scroll" specifically (not just "↓") to avoid false positives
	// from navigation arrows that may appear in the footer.
	assert.Contains(t, stripped, "scroll", "expected scroll hint in output: %q", stripped)
}

func TestRenderAnalytics_ScrollOffsetClampsToMax(t *testing.T) {
	app := makeAnalyticsApp()
	app.height = 10
	app.analyticsMode = true
	app.analyticsScrollOffset = 9999 // way past end
	app.recommendations = []model.Recommendation{
		{Severity: model.SeverityWarning, Category: model.CategoryShardHealth, Title: "Item A"},
	}

	// View-side clamping: renderAnalytics must never panic and must show content
	// despite the out-of-range stored offset (clamped locally for display only).
	result := renderAnalytics(app)
	assert.NotEmpty(t, result)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "Item A", "view-side clamping should still show recommendation content")
}

// TestApp_WindowResizeClampsScrollOffset verifies that a terminal resize event
// (WindowSizeMsg) clamps analyticsScrollOffset to the new content maximum so
// subsequent CursorUp presses respond immediately without working off stale debt.
func TestApp_WindowResizeClampsScrollOffset(t *testing.T) {
	app := makeAnalyticsApp()
	app.analyticsMode = true
	app.recommendations = []model.Recommendation{
		{Severity: model.SeverityWarning, Category: model.CategoryShardHealth, Title: "Item A"},
	}
	// Set an offset far beyond what a small terminal allows.
	app.analyticsScrollOffset = 9999

	// Simulate a terminal resize.
	newModel, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = newModel.(*App)

	max := analyticsMaxOffset(app)
	assert.LessOrEqual(t, app.analyticsScrollOffset, max,
		"resize must clamp stored offset to the real content maximum")
}

// TestApp_AnalyticsModeToggle verifies that pressing 'a' enters and exits analytics mode.
func TestApp_AnalyticsModeToggle(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	require.False(t, app.analyticsMode)

	// Press 'a' to enter analytics mode.
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	app = newModel.(*App)
	assert.True(t, app.analyticsMode)
	assert.Equal(t, 0, app.analyticsScrollOffset)

	// Press 'a' again to exit analytics mode — offset is reset.
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	app = newModel.(*App)
	assert.False(t, app.analyticsMode)
	assert.Equal(t, 0, app.analyticsScrollOffset)

	// Re-entering resets offset even after prior scroll.
	app.analyticsScrollOffset = 5
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	app = newModel.(*App)
	assert.True(t, app.analyticsMode)
	assert.Equal(t, 0, app.analyticsScrollOffset, "re-entering analytics mode must reset scroll offset")
}

// TestApp_EscExitsAnalyticsMode verifies that pressing esc exits analytics mode and resets scroll offset.
func TestApp_EscExitsAnalyticsMode(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.analyticsMode = true
	app.analyticsScrollOffset = 7

	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = newModel.(*App)
	assert.False(t, app.analyticsMode)
	assert.Equal(t, 0, app.analyticsScrollOffset)
}

// TestApp_AnalyticsModeScrolling verifies ↑↓ scrolling while in analytics mode.
func TestApp_AnalyticsModeScrolling(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 80
	app.height = 12 // small enough to force overflow with the recommendations below
	app.analyticsMode = true
	// 4 categories × 3 items = 20 content lines; with height=12 maxOffset ≈ 12,
	// so starting at offset 5 and scrolling down to 6 is within valid range.
	app.recommendations = []model.Recommendation{
		{Severity: model.SeverityWarning, Category: model.CategoryResourcePressure, Title: "CPU high A"},
		{Severity: model.SeverityWarning, Category: model.CategoryResourcePressure, Title: "CPU high B"},
		{Severity: model.SeverityWarning, Category: model.CategoryResourcePressure, Title: "CPU high C"},
		{Severity: model.SeverityWarning, Category: model.CategoryShardHealth, Title: "Shard A"},
		{Severity: model.SeverityWarning, Category: model.CategoryShardHealth, Title: "Shard B"},
		{Severity: model.SeverityWarning, Category: model.CategoryShardHealth, Title: "Shard C"},
		{Severity: model.SeverityWarning, Category: model.CategoryIndexConfig, Title: "Index A"},
		{Severity: model.SeverityWarning, Category: model.CategoryIndexConfig, Title: "Index B"},
		{Severity: model.SeverityWarning, Category: model.CategoryIndexConfig, Title: "Index C"},
		{Severity: model.SeverityWarning, Category: model.CategoryHotspot, Title: "Hot A"},
		{Severity: model.SeverityWarning, Category: model.CategoryHotspot, Title: "Hot B"},
		{Severity: model.SeverityWarning, Category: model.CategoryHotspot, Title: "Hot C"},
	}
	app.analyticsScrollOffset = 5

	// Scroll down.
	newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = newModel.(*App)
	assert.Equal(t, 6, app.analyticsScrollOffset)

	// Scroll up.
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	app = newModel.(*App)
	assert.Equal(t, 5, app.analyticsScrollOffset)

	// Scroll up at zero should clamp.
	app.analyticsScrollOffset = 0
	newModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	app = newModel.(*App)
	assert.Equal(t, 0, app.analyticsScrollOffset)
}

// TestApp_AnalyticsModeScrollDownCap verifies that pressing ↓ many times clamps
// analyticsScrollOffset to the real content maximum. Without clamping, the stored
// offset can accumulate overscroll debt: pressing ↑ would decrement the stored
// value but the display would remain frozen at the bottom until the debt is paid.
func TestApp_AnalyticsModeScrollDownCap(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 100
	app.height = 30
	app.analyticsMode = true
	app.recommendations = []model.Recommendation{
		{Severity: model.SeverityWarning, Category: model.CategoryShardHealth, Title: "Item A"},
		{Severity: model.SeverityWarning, Category: model.CategoryShardHealth, Title: "Item B"},
	}

	// Press ↓ far more times than there are content lines.
	const presses = 500
	for i := 0; i < presses; i++ {
		newModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyDown})
		app = newModel.(*App)
	}

	// Stored offset must be clamped to the real content maximum, not the raw
	// press count. With only 2 items the content fits within the terminal
	// (width=100, height=30), so the maximum scroll offset is 0.
	assert.Equal(t, 0, app.analyticsScrollOffset,
		"stored offset should be clamped to the real content max, not the press count")
}

// TestApp_AnalyticsModeViewContainsTitle verifies View() renders analytics title bar.
func TestApp_AnalyticsModeViewContainsTitle(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 100
	app.height = 30
	app.current = &model.Snapshot{FetchedAt: time.Now()}
	app.current.Health.Status = "green"
	app.analyticsMode = true
	app.recommendations = []model.Recommendation{}

	view := app.View()
	stripped := stripANSI(view)
	assert.Contains(t, stripped, "Analytics")
	assert.Contains(t, stripped, "No issues found")
	// Dashboard elements should NOT appear in analytics mode.
	assert.NotContains(t, stripped, "Indices")
}

// TestApp_ViewDoesNotContainAnalyticsWhenModeOff verifies normal view is shown
// when analyticsMode is false.
func TestApp_ViewDoesNotContainAnalyticsWhenModeOff(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 100
	app.height = 30
	app.current = &model.Snapshot{FetchedAt: time.Now()}
	app.current.Health.Status = "green"
	app.current.Health.NumberOfNodes = 3
	app.analyticsMode = false

	view := app.View()
	stripped := stripANSI(view)
	// Normal view should show cluster info, not analytics.
	assert.NotContains(t, stripped, "Analytics — Cluster Recommendations")
}
