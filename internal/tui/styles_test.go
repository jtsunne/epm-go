package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestSetTheme(t *testing.T) {
	orig := lipgloss.HasDarkBackground()
	t.Cleanup(func() { lipgloss.SetHasDarkBackground(orig) })

	tests := []struct {
		theme    string
		wantErr  bool
		wantDark *bool
	}{
		{theme: "light", wantDark: ptr(false)},
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

func ptr[T any](v T) *T { return &v }
