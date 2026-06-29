package tui

import (
	"fmt"
	"strings"

	"sattchel/internal/optimizely/core"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

// RenderFlagLipGloss renders a FeatureFlagDefinition and its instances as
// styled tabular output using lipgloss tables.
func RenderFlagLipGloss(flag *core.FeatureFlagDefinition, instances []core.FeatureFlagInstance) error {
	s := AutoStyles()

	// ── Header ──────────────────────────────────────────────────────────
	status := "✅ Active"
	if flag.Archived {
		status = "🗄️ Archived"
	}

	fmt.Println(s.Title.Render("🚩 " + flag.Name))
	fmt.Println(s.Muted.Render("  " + flag.Key + " — " + status))
	fmt.Println()

	if flag.Description != "" {
		fmt.Println(s.Text.Render("  " + flag.Description))
		fmt.Println()
	}

	// ── Details ─────────────────────────────────────────────────────────
	fmt.Println(s.Title.Render("  DETAILS"))

	detailPairs := [][]string{
		{"ID", flag.ID},
		{"Archived", fmt.Sprintf("%t", flag.Archived)},
	}
	if flag.CreatedAt != nil {
		detailPairs = append(detailPairs, []string{"Created", flag.CreatedAt.Format("2006-01-02")})
	}
	if flag.CreatedBy != nil {
		detailPairs = append(detailPairs, []string{"Created By", *flag.CreatedBy})
	}
	fmt.Println(renderTable(s, detailPairs))
	fmt.Println()

	// ── Default Variables ───────────────────────────────────────────────
	if hasVariables(flag.DefaultVariables) {
		fmt.Println(s.Title.Render("  DEFAULT VARIABLES"))
		varRows := [][]string{
			{"Variable", "Type", "Value", "Description"},
		}
		for key, v := range flag.DefaultVariables.BoolVariables {
			varRows = append(varRows, []string{key, "boolean", fmt.Sprintf("%v", v.Value), v.Description})
		}
		for key, v := range flag.DefaultVariables.IntVariables {
			varRows = append(varRows, []string{key, "integer", fmt.Sprintf("%v", v.Value), v.Description})
		}
		for key, v := range flag.DefaultVariables.FloatVariables {
			varRows = append(varRows, []string{key, "float", fmt.Sprintf("%v", v.Value), v.Description})
		}
		for key, v := range flag.DefaultVariables.StringVariables {
			varRows = append(varRows, []string{key, "string", v.Value, v.Description})
		}
		for key, v := range flag.DefaultVariables.JsonVariables {
			varRows = append(varRows, []string{key, "json", fmt.Sprintf("%v", v.Value), v.Description})
		}
		fmt.Println(renderTable(s, varRows))
		fmt.Println()
	}

	// ── Environment Overrides ───────────────────────────────────────────
	if len(instances) > 0 {
		fmt.Println(s.Title.Render("  ENVIRONMENT OVERRIDES"))
		for i, inst := range instances {
			fmt.Println(s.Text.Bold(true).Render("  " + inst.EnvironmentID))

			instDetails := [][]string{
				{"Enabled", enabledStr(inst.Enabled)},
				{"Archived", fmt.Sprintf("%t", inst.Archived)},
			}
			fmt.Println(renderTable(s, instDetails))

			if hasVariables(inst.Variables) {
				fmt.Println()
				fmt.Println(s.Info.Bold(true).Render("  Overrides"))
				varRows := [][]string{
					{"Variable", "Type", "Value", "Description"},
				}
				for key, v := range inst.Variables.BoolVariables {
					varRows = append(varRows, []string{key, "boolean", fmt.Sprintf("%v", v.Value), v.Description})
				}
				for key, v := range inst.Variables.IntVariables {
					varRows = append(varRows, []string{key, "integer", fmt.Sprintf("%v", v.Value), v.Description})
				}
				for key, v := range inst.Variables.FloatVariables {
					varRows = append(varRows, []string{key, "float", fmt.Sprintf("%v", v.Value), v.Description})
				}
				for key, v := range inst.Variables.StringVariables {
					varRows = append(varRows, []string{key, "string", v.Value, v.Description})
				}
				for key, v := range inst.Variables.JsonVariables {
					varRows = append(varRows, []string{key, "json", fmt.Sprintf("%v", v.Value), v.Description})
				}
				fmt.Println(renderTable(s, varRows))
			}

			if i < len(instances)-1 {
				fmt.Println()
				fmt.Println(strings.Repeat("─", 40))
				fmt.Println()
			}
		}
	}

	return nil
}

// renderTable renders a table using the bubbles/table component.
func renderTable(s Styles, rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	// First row is the header
	header := rows[0]
	data := rows[1:]

	columns := make([]table.Column, len(header))
	for i, title := range header {
		columns[i] = table.Column{
			Title: title,
			Width: 20,
		}
	}

	rowData := make([]table.Row, len(data))
	for i, r := range data {
		rowData[i] = r
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rowData),
		table.WithFocused(false),
		table.WithHeight(len(data)+2),
		table.WithWidth(80),
	)

	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Bold(true)
	styles.Cell = styles.Cell.
		Foreground(s.Text.GetForeground())
	t.SetStyles(styles)

	return t.View()
}

// hasVariables returns true if any variable type has entries.
func hasVariables(vars core.Variables) bool {
	return len(vars.BoolVariables) > 0 ||
		len(vars.IntVariables) > 0 ||
		len(vars.FloatVariables) > 0 ||
		len(vars.StringVariables) > 0 ||
		len(vars.JsonVariables) > 0
}

// enabledStr returns a human-readable string for the enabled state.
func enabledStr(enabled bool) string {
	if enabled {
		return "✅ Yes"
	}
	return "❌ No"
}
