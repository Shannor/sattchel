package driving

import (
	"fmt"
	"strings"

	"sattchel/internal/tui"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
)

// renderConfig formats the Configuration as a styled string for terminal output.
func renderConfig(cfg *Configuration, styles tui.Styles) string {

	enumeratorStyle := lipgloss.NewStyle().Foreground(styles.Success.GetForeground()).MarginRight(1)
	rootStyle := lipgloss.NewStyle().Foreground(styles.Title.GetForeground())
	itemStyle := lipgloss.NewStyle().Foreground(styles.Text.GetForeground())

	t := tree.Root(styles.Title.Render("Optimizely Configs"))

	if cfg.APIKey == "" {
		t = t.Child(
			"API Key",
			tree.New().Child(
				styles.Muted.Render("(not set)"),
			),
		)
	} else {
		maskedKey := maskString(cfg.APIKey)
		t = t.Child(
			"API Key",
			tree.New().Child(
				styles.Muted.Render(maskedKey),
			),
		)
	}
	t = t.Child("Projects")
	if len(cfg.Projects) == 0 {
		t = t.Child(tree.New().Child("(none)"))
	} else {
		children := tree.New().Child()
		for _, p := range cfg.Projects {
			status := styles.Muted.Render("inactive")
			if p.IsActive {
				status = styles.Success.Render("active")
			}
			children = children.Child(
				fmt.Sprintf("%s — %s (%s)",
					styles.Text.Render(p.Name),
					styles.Muted.Render(p.ID),
					status,
				),
			)
		}
		t = t.Child(children)
	}

	return t.
		Enumerator(tree.RoundedEnumerator).
		EnumeratorStyle(enumeratorStyle).
		RootStyle(rootStyle).
		ItemStyle(itemStyle).
		String()
}

func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}
