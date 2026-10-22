package tui

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestSetTheme(t *testing.T) {
	orig := lipgloss.HasDarkBackground()
	t.Cleanup(func() {
		_ = SetTheme("auto")
		lipgloss.SetHasDarkBackground(orig)
	})

	tests := []struct {
		theme    string
		wantErr  bool
		wantDark *bool
	}{
		{theme: "light", wantDark: ptr(false)},
		{theme: "latte", wantDark: ptr(false)},
		{theme: "frappe", wantDark: ptr(true)},
		{theme: "macchiato", wantDark: ptr(true)},
		{theme: "mocha", wantDark: ptr(true)},
		{theme: "dark", wantDark: ptr(true)},
		{theme: "auto"},
		{theme: ""},
		{theme: "solarized", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.theme, func(t *testing.T) {
			err := SetTheme(tt.theme)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SetTheme(%q) error = %v, wantErr %v", tt.theme, err, tt.wantErr)
			}
			if tt.wantDark != nil && lipgloss.HasDarkBackground() != *tt.wantDark {
				t.Errorf("SetTheme(%q): HasDarkBackground = %v, want %v", tt.theme, lipgloss.HasDarkBackground(), *tt.wantDark)
			}
		})
	}
}

func TestAutoAndExplicitThemeBackgroundSelection(t *testing.T) {
	orig := lipgloss.HasDarkBackground()
	t.Cleanup(func() {
		_ = SetTheme("auto")
		lipgloss.SetHasDarkBackground(orig)
	})

	// Auto resets explicit overrides and re-detects the real terminal.
	// Headless tests must not assume its light/dark value.
	lipgloss.SetHasDarkBackground(false)
	if err := SetTheme("auto"); err != nil {
		t.Fatal(err)
	}
	if hasCustomTheme || colorBase != nil {
		t.Fatal("auto must restore the adaptive palette without a custom background")
	}
	if err := SetTheme("mocha"); err != nil {
		t.Fatal(err)
	}
	if !lipgloss.HasDarkBackground() {
		t.Fatal("explicit mocha theme should select a dark palette independent of terminal detection")
	}
	if err := SetTheme("latte"); err != nil {
		t.Fatal(err)
	}
	if lipgloss.HasDarkBackground() {
		t.Fatal("explicit latte theme should select a light palette independent of terminal detection")
	}
}

func ptr[T any](v T) *T { return &v }

func TestCatppuccinPalettes(t *testing.T) {
	orig := lipgloss.HasDarkBackground()
	t.Cleanup(func() {
		_ = SetTheme("auto")
		lipgloss.SetHasDarkBackground(orig)
	})
	for _, theme := range []string{"latte", "frappe", "macchiato", "mocha"} {
		t.Run(theme, func(t *testing.T) {
			if err := SetTheme(theme); err != nil {
				t.Fatal(err)
			}
			p := catppuccinThemes[theme]
			if got := colorFg.Dark; got != p.fg {
				t.Errorf("foreground = %s, want %s", got, p.fg)
			}
			if colorBase == nil {
				t.Fatal("custom theme should set an explicit background")
			}
		})
	}
}

func TestCatppuccinSemanticPaletteMappings(t *testing.T) {
	// Representative official palette values from Catppuccin palette v1.8.0.
	want := map[string]themeColors{
		"latte":     {green: "#40a02b", yellow: "#df8e1d", red: "#d20f39", blue: "#1e66f5", fg: "#4c4f69", surface: "#ccd0da", base: "#eff1f5"},
		"frappe":    {green: "#a6d189", yellow: "#e5c890", red: "#e78284", blue: "#8caaee", fg: "#c6d0f5", surface: "#414559", base: "#303446"},
		"macchiato": {green: "#a6da95", yellow: "#eed49f", red: "#ed8796", blue: "#8aadf4", fg: "#cad3f5", surface: "#363a4f", base: "#24273a"},
		"mocha":     {green: "#a6e3a1", yellow: "#f9e2af", red: "#f38ba8", blue: "#89b4fa", fg: "#cdd6f4", surface: "#313244", base: "#1e1e2e"},
	}
	for theme, expected := range want {
		got := catppuccinThemes[theme]
		if got.green != expected.green || got.yellow != expected.yellow || got.red != expected.red || got.blue != expected.blue || got.fg != expected.fg || got.surface != expected.surface || got.base != expected.base {
			t.Errorf("%s palette mappings differ: got %+v, want representative values %+v", theme, got, expected)
		}
		for _, bg := range []string{got.base, got.surface, got.cursor, got.marked} {
			if ratio := contrastRatio(got.fg, bg); ratio < 4.5 {
				t.Errorf("%s primary text %s on background %s has contrast %.2f:1, want at least 4.5:1", theme, got.fg, bg, ratio)
			}
		}
	}
}

func contrastRatio(fg, bg string) float64 {
	return (math.Max(relativeLuminance(fg), relativeLuminance(bg)) + 0.05) / (math.Min(relativeLuminance(fg), relativeLuminance(bg)) + 0.05)
}

func relativeLuminance(hex string) float64 {
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
		panic(err)
	}
	linear := func(channel int) float64 {
		c := float64(channel) / 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}

func TestThemeSwitchRefreshesCachedStylesAndANSI(t *testing.T) {
	origBackground := lipgloss.HasDarkBackground()
	origProfile := lipgloss.ColorProfile()
	t.Cleanup(func() {
		_ = SetTheme("auto")
		lipgloss.SetHasDarkBackground(origBackground)
		lipgloss.SetColorProfile(origProfile)
	})
	lipgloss.SetColorProfile(termenv.TrueColor)

	for _, theme := range []string{"latte", "mocha", "frappe", "macchiato", "mocha"} {
		if err := SetTheme(theme); err != nil {
			t.Fatal(err)
		}
		got := StyleHeader.Render("theme")
		if !strings.Contains(got, "\x1b[38;2;") || !strings.Contains(got, "\x1b[48;2;") {
			t.Errorf("%s header should render true-color foreground and background ANSI: %q", theme, got)
		}
		p := catppuccinThemes[theme]
		for _, want := range []string{rgbANSIPrefix(38, p.fg), rgbANSIPrefix(48, p.header)} {
			if !strings.Contains(got, want) {
				t.Errorf("%s header ANSI missing %q: %q", theme, want, got)
			}
		}
	}
	if err := SetTheme("mocha"); err != nil {
		t.Fatal(err)
	}
	mochaHeader := StyleHeader.Render("theme")
	if err := SetTheme("latte"); err != nil {
		t.Fatal(err)
	}
	if got := StyleHeader.Render("theme"); got == mochaHeader {
		t.Fatal("cached header style did not change after switching themes")
	}
}

func rgbANSIPrefix(code int, hex string) string {
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%d;2;%d;%d;", code, r, g)
}

func TestInvalidThemeLeavesActiveThemeUnchanged(t *testing.T) {
	orig := lipgloss.HasDarkBackground()
	t.Cleanup(func() {
		_ = SetTheme("auto")
		lipgloss.SetHasDarkBackground(orig)
	})
	if err := SetTheme("mocha"); err != nil {
		t.Fatal(err)
	}
	before := StyleHeader.Render("active")
	if err := SetTheme("solarized"); err == nil || !strings.Contains(err.Error(), "invalid theme") {
		t.Fatalf("SetTheme(solarized) error = %v, want meaningful invalid-theme error", err)
	}
	if got := StyleHeader.Render("active"); got != before {
		t.Fatal("invalid theme changed the active cached styles")
	}
}

func TestCursorDistinctFromSurface(t *testing.T) {
	for name, p := range catppuccinThemes {
		if p.cursor == p.surface {
			t.Errorf("%s cursor indistinguishable from zebra row", name)
		}
		if ratio := contrastRatio(p.fg, p.cursor); ratio < 4.5 {
			t.Errorf("%s cursor text contrast %.2f:1", name, ratio)
		}
	}
}

func TestLatteRedStatusContrast(t *testing.T) {
	if err := SetTheme("latte"); err != nil { t.Fatal(err) }
	t.Cleanup(func() { _ = SetTheme("auto") })
	fg, ok := statusCardForeground("red").(lipgloss.Color)
	if !ok { t.Fatal("expected explicit status foreground") }
	if ratio := contrastRatio(string(fg), catppuccinThemes["latte"].red); ratio < 4.5 {
		t.Fatalf("Latte red status contrast %.2f:1", ratio)
	}
}

func TestAutoResetsExplicitRendererOverride(t *testing.T) {
	if err := SetTheme("latte"); err != nil { t.Fatal(err) }
	if err := SetTheme("auto"); err != nil { t.Fatal(err) }
	if hasCustomTheme { t.Fatal("auto must clear custom theme") }
	if colorBase != nil { t.Fatal("auto must clear custom background") }
	// A fresh default renderer must no longer retain Latte's forced light setting.
	// Actual terminal detection is environment-dependent and is not asserted here.
}
