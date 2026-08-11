package tui

import (
	"image/color"
	"os"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// ThemeType represents the name of a TUI theme.
type ThemeType string

const (
	// ThemeDefault is the default theme (Dracula/Charm colors in dark mode, light/subtle colors in light mode).
	ThemeDefault ThemeType = "default"
	// ThemeGruvbox is the Gruvbox color scheme.
	ThemeGruvbox ThemeType = "gruvbox"
)

var currentTheme ThemeType = ThemeDefault

// SetTheme sets the active theme for the TUI components.
func SetTheme(theme string) {
	switch ThemeType(theme) {
	case ThemeGruvbox:
		currentTheme = ThemeGruvbox
	default:
		currentTheme = ThemeDefault
	}
}

// GetTheme returns the currently configured theme.
func GetTheme() ThemeType {
	return currentTheme
}

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

// AutoStyles creates the styles by automatically detecting the terminal's background
// and using the active theme.
func AutoStyles() Styles {
	hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
	return GetStyles(currentTheme, hasDark)
}

// GetStyles returns a Styles struct given the theme and whether the terminal is dark or light.
func GetStyles(theme ThemeType, isDark bool) Styles {
	switch theme {
	case ThemeGruvbox:
		return GruvboxStyles(isDark)
	default:
		return DefaultStyles(isDark)
	}
}

// DefaultStyles returns a pre-configured Styles struct for the default theme given whether the terminal is dark or light.
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

// GruvboxStyles returns a pre-configured Styles struct for the Gruvbox theme given whether the terminal is dark or light.
func GruvboxStyles(isDark bool) Styles {
	ld := lipgloss.LightDark(isDark)

	// Gruvbox fg1: light #3c3836, dark #ebdbb2
	textCol := ld(lipgloss.Color("#3c3836"), lipgloss.Color("#ebdbb2"))

	// Gruvbox Red: light #cc241d (neutral red), dark #fb4934 (bright red)
	errorCol := ld(lipgloss.Color("#cc241d"), lipgloss.Color("#fb4934"))

	// Gruvbox Green: light #98971a (neutral green), dark #b8bb26 (bright green)
	successCol := ld(lipgloss.Color("#98971a"), lipgloss.Color("#b8bb26"))

	// Gruvbox Orange: light #d65d0e (neutral orange), dark #fe8019 (bright orange)
	warningCol := ld(lipgloss.Color("#d65d0e"), lipgloss.Color("#fe8019"))

	// Gruvbox Blue: light #458588 (neutral blue), dark #83a598 (bright blue)
	infoCol := ld(lipgloss.Color("#458588"), lipgloss.Color("#83a598"))

	// Gruvbox Gray/fg4: light #928374 (gray), dark #a89984 (fg4)
	mutedCol := ld(lipgloss.Color("#928374"), lipgloss.Color("#a89984"))

	// Gruvbox Title: light bg #ebdbb2 (bg1), fg #282828 (bg0); dark bg #3c3836 (bg1), fg #fbf1c7 (fg0)
	titleBg := ld(lipgloss.Color("#ebdbb2"), lipgloss.Color("#3c3836"))
	titleFg := ld(lipgloss.Color("#282828"), lipgloss.Color("#fbf1c7"))

	return Styles{
		Text: lipgloss.NewStyle().Foreground(textCol),
		Error: lipgloss.NewStyle().
			Foreground(errorCol).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(successCol),
		Warning: lipgloss.NewStyle().
			Foreground(warningCol),
		Info: lipgloss.NewStyle().
			Foreground(infoCol),
		Muted: lipgloss.NewStyle().
			Foreground(mutedCol),
		Title: lipgloss.NewStyle().
			Background(titleBg).
			Foreground(titleFg).
			Padding(0, 1).
			Bold(true),
	}
}

// ThemeDefaultStyles returns a new huh.Styles configured for the default theme.
func ThemeDefaultStyles(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)

	var bg, fg, muted, red, green, blue, orange, selection color.Color

	if isDark {
		bg = lipgloss.Color("#282A36")
		fg = lipgloss.Color("#F8F8F2")
		muted = lipgloss.Color("#94A3B8")
		red = lipgloss.Color("#FF5555")
		green = lipgloss.Color("#50FA7B")
		blue = lipgloss.Color("#8BE9FD")
		orange = lipgloss.Color("#FFB86C")
		selection = lipgloss.Color("#44475A")
	} else {
		bg = lipgloss.Color("#F8F9FA")
		fg = lipgloss.Color("#212529")
		muted = lipgloss.Color("#6C757D")
		red = lipgloss.Color("#D90429")
		green = lipgloss.Color("#008000")
		blue = lipgloss.Color("#0077B6")
		orange = lipgloss.Color("#E67E22")
		selection = lipgloss.Color("#DEE2E6")
	}

	t.Focused.Base = t.Focused.Base.BorderForeground(selection)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(blue).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(blue).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(muted)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.Directory = t.Focused.Directory.Foreground(blue)
	t.Focused.File = t.Focused.File.Foreground(fg)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(orange)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(orange)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(orange)
	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(orange)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(orange).Bold(true)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(muted)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(bg).Background(orange).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(fg).Background(selection)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(orange)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(muted)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(orange)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

// ThemeGruvboxStyles returns a new huh.Styles configured for the Gruvbox theme.
func ThemeGruvboxStyles(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)

	var bg, fg, gray, red, green, blue, orange, selection color.Color

	if isDark {
		bg = lipgloss.Color("#282828")        // bg0
		fg = lipgloss.Color("#ebdbb2")        // fg1
		gray = lipgloss.Color("#928374")      // gray
		red = lipgloss.Color("#fb4934")       // bright red
		green = lipgloss.Color("#b8bb26")     // bright green
		blue = lipgloss.Color("#83a598")      // bright blue
		orange = lipgloss.Color("#fe8019")    // bright orange
		selection = lipgloss.Color("#504945") // bg2
	} else {
		bg = lipgloss.Color("#fbf1c7")        // bg0
		fg = lipgloss.Color("#3c3836")        // fg1
		gray = lipgloss.Color("#928374")      // gray
		red = lipgloss.Color("#cc241d")       // neutral red
		green = lipgloss.Color("#98971a")     // neutral green
		blue = lipgloss.Color("#458588")      // neutral blue
		orange = lipgloss.Color("#d65d0e")    // neutral orange
		selection = lipgloss.Color("#d5c4a1") // bg2
	}

	t.Focused.Base = t.Focused.Base.BorderForeground(selection)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(blue).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(blue).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(gray)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.Directory = t.Focused.Directory.Foreground(blue)
	t.Focused.File = t.Focused.File.Foreground(fg)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(orange)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(orange)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(orange)
	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(orange)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(orange).Bold(true)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(gray)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(bg).Background(orange).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(fg).Background(selection)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(orange)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(gray)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(orange)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}

// HuhTheme returns the appropriate huh.Theme for the active theme.
func HuhTheme() huh.Theme {
	switch currentTheme {
	case ThemeGruvbox:
		return huh.ThemeFunc(ThemeGruvboxStyles)
	default:
		return huh.ThemeFunc(ThemeDefaultStyles)
	}
}

// NewForm returns a new *huh.Form with the current theme applied.
func NewForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(HuhTheme())
}
