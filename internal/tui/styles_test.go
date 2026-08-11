package tui

import (
	"strconv"
	"testing"
)

func TestThemeSelection(t *testing.T) {
	// Clean up at the end of the test to avoid leaking global state
	defer SetTheme("default")

	tests := []struct {
		input    string
		expected ThemeType
	}{
		{"default", ThemeDefault},
		{"gruvbox", ThemeGruvbox},
		{"invalid", ThemeDefault}, // fallback
		{"", ThemeDefault},        // fallback
	}

	for _, tc := range tests {
		SetTheme(tc.input)
		if GetTheme() != tc.expected {
			t.Errorf("expected theme %q for input %q, got %q", tc.expected, tc.input, GetTheme())
		}
	}
}

func TestGetStyles(t *testing.T) {
	// Test that both default and gruvbox styles can be loaded for dark and light modes
	themes := []ThemeType{ThemeDefault, ThemeGruvbox}
	backgrounds := []bool{true, false}

	for _, theme := range themes {
		for _, isDark := range backgrounds {
			t.Run(string(theme)+"_dark_"+strconv.FormatBool(isDark), func(t *testing.T) {
				s := GetStyles(theme, isDark)

				// Assert that essential style foregrounds are set
				if s.Text.GetForeground() == nil {
					t.Error("expected Text foreground color to be set")
				}
				if s.Error.GetForeground() == nil {
					t.Error("expected Error foreground color to be set")
				}
				if s.Success.GetForeground() == nil {
					t.Error("expected Success foreground color to be set")
				}
				if s.Warning.GetForeground() == nil {
					t.Error("expected Warning foreground color to be set")
				}
				if s.Info.GetForeground() == nil {
					t.Error("expected Info foreground color to be set")
				}
				if s.Muted.GetForeground() == nil {
					t.Error("expected Muted foreground color to be set")
				}
			})
		}
	}
}

func TestHuhTheme(t *testing.T) {
	defer SetTheme("default")

	for _, theme := range []string{"default", "gruvbox"} {
		SetTheme(theme)
		hTheme := HuhTheme()
		if hTheme == nil {
			t.Errorf("expected non-nil huh.Theme for theme %s", theme)
			continue
		}

		// Ensure the returned theme function works for both dark and light modes
		stylesDark := hTheme.Theme(true)
		if stylesDark == nil {
			t.Errorf("expected non-nil styles for dark mode in theme %s", theme)
		}

		stylesLight := hTheme.Theme(false)
		if stylesLight == nil {
			t.Errorf("expected non-nil styles for light mode in theme %s", theme)
		}
	}
}
