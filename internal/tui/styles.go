package tui

import (
	"os"

	"charm.land/lipgloss/v2"
)

// Styles holds the common Lip Gloss styles used across the TUI.
type Styles struct {
	Text    lipgloss.Style
	Error   lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Muted   lipgloss.Style
	Title   lipgloss.Style
}

// AutoStyles creates the default styles by automatically detecting the terminal's background.
func AutoStyles() Styles {
	hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	return DefaultStyles(hasDark)
}

// DefaultStyles returns a pre-configured Styles struct given whether the terminal is dark or light.
func DefaultStyles(isDark bool) Styles {
	ld := lipgloss.LightDark(isDark)

	return Styles{
		// Standard text
		Text: lipgloss.NewStyle().Foreground(ld(lipgloss.Color("#000000"), lipgloss.Color("#FFFFFF"))),

		// Error text (Red)
		Error: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#D90429"), lipgloss.Color("#FF5555"))).
			Bold(true),

		// Success text (Green)
		Success: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#008000"), lipgloss.Color("#50FA7B"))),

		// Warning text (Orange/Yellow)
		Warning: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#E67E22"), lipgloss.Color("#FFB86C"))),

		// Info text (Blue/Cyan)
		Info: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#0077B6"), lipgloss.Color("#8BE9FD"))),

		// Muted/Subtle text (Gray)
		Muted: lipgloss.NewStyle().
			Foreground(ld(lipgloss.Color("#6C757D"), lipgloss.Color("#6272A4"))),

		// Title blocks with a background
		Title: lipgloss.NewStyle().
			Background(ld(lipgloss.Color("#E9ECEF"), lipgloss.Color("#44475A"))).
			Foreground(ld(lipgloss.Color("#212529"), lipgloss.Color("#F8F8F2"))).
			Padding(0, 1).
			Bold(true),
	}
}
