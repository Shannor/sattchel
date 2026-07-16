package tui

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// RenderTable renders a styled table with the given headers and rows.
// It automatically calculates column widths and applies the Sattchel TUI style.
func RenderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 && len(rows) == 0 {
		return ""
	}

	s := AutoStyles()

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers(headers...)

	for _, row := range rows {
		t.Row(row...)
	}

	t.StyleFunc(func(row, col int) lipgloss.Style {
		style := lipgloss.NewStyle().Padding(0, 1)

		// Header styling
		if row == table.HeaderRow {
			return style.
				Bold(true).
				Foreground(s.Title.GetForeground()).
				Background(s.Title.GetBackground())
		}

		// Row cells styling
		return style.Foreground(s.Text.GetForeground())
	})

	return t.String()
}
