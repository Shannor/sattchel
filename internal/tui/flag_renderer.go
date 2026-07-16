package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"sattchel/internal/optimizely/core"
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

	detailHeaders := []string{"Field", "Value"}
	detailRows := [][]string{
		{"ID", flag.ID},
		{"Archived", fmt.Sprintf("%t", flag.Archived)},
	}
	if flag.CreatedAt != nil {
		detailRows = append(detailRows, []string{"Created", flag.CreatedAt.Format("2006-01-02")})
	}
	if flag.CreatedBy != nil {
		detailRows = append(detailRows, []string{"Created By", *flag.CreatedBy})
	}
	fmt.Println(RenderTable(detailHeaders, detailRows))
	fmt.Println()

	// ── Default Variables ───────────────────────────────────────────────
	if hasVariables(flag.DefaultVariables) {
		fmt.Println(s.Title.Render("  DEFAULT VARIABLES"))
		varHeaders := []string{"Variable", "Type", "Value", "Description"}
		var varRows [][]string
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
		fmt.Println(RenderTable(varHeaders, varRows))
		fmt.Println()
	}

	// ── Environment Overrides ───────────────────────────────────────────
	if len(instances) > 0 {
		fmt.Println(s.Title.Render("  ENVIRONMENT OVERRIDES"))
		for i, inst := range instances {
			fmt.Println(s.Text.Bold(true).Render("  " + inst.EnvironmentID))

			instHeaders := []string{"Field", "Value"}
			instRows := [][]string{
				{"Enabled", enabledStr(inst.Enabled)},
				{"Archived", fmt.Sprintf("%t", inst.Archived)},
			}
			fmt.Println(RenderTable(instHeaders, instRows))

			if hasVariables(inst.Variables) {
				fmt.Println()
				fmt.Println(s.Info.Bold(true).Render("  Overrides"))
				varHeaders := []string{"Variable", "Type", "Value", "Description"}
				var varRows [][]string
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
				fmt.Println(RenderTable(varHeaders, varRows))
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

// RenderMultiProjectFlagLipGloss renders multiple ProjectFlagReports using LipGloss styling and tables.
func RenderMultiProjectFlagLipGloss(reports []ProjectFlagReport, opts ReportOptions) error {
	out, err := RenderMultiProjectFlagLipGlossStr(reports, opts)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// RenderMultiProjectFlagLipGlossStr renders multiple ProjectFlagReports using LipGloss styling and tables, returning the formatted string.
func RenderMultiProjectFlagLipGlossStr(reports []ProjectFlagReport, opts ReportOptions) (string, error) {
	if len(reports) == 0 {
		return "No data available.\n", nil
	}

	var sb strings.Builder
	s := AutoStyles()

	firstReport := reports[0]
	sb.WriteString(s.Title.Render("🚩 "+firstReport.Flag.Name) + "\n")
	sb.WriteString(s.Muted.Render("  "+firstReport.Flag.Key) + "\n")
	sb.WriteString("\n")

	if firstReport.Flag.Description != "" {
		sb.WriteString(s.Text.Render("  "+firstReport.Flag.Description) + "\n")
		sb.WriteString("\n")
	}

	for idx, rep := range reports {
		// Project Header
		sb.WriteString(s.Title.Render(fmt.Sprintf("🏢 PROJECT: %s (%s)", strings.ToUpper(rep.Project.Name), rep.Project.ID)) + "\n")
		sb.WriteString("\n")

		// Details
		if opts.ShowDetails {
			sb.WriteString(s.Info.Bold(true).Render("  Details") + "\n")
			status := "✅ Active"
			if rep.Flag.Archived {
				status = "🗄️ Archived"
			}
			detailHeaders := []string{"Field", "Value"}
			detailRows := [][]string{
				{"Status", status},
				{"ID", rep.Flag.ID},
				{"Archived", fmt.Sprintf("%t", rep.Flag.Archived)},
			}
			if rep.Flag.CreatedAt != nil {
				detailRows = append(detailRows, []string{"Created", rep.Flag.CreatedAt.Format("2006-01-02")})
			}
			if rep.Flag.CreatedBy != nil {
				detailRows = append(detailRows, []string{"Created By", *rep.Flag.CreatedBy})
			}
			sb.WriteString(RenderTable(detailHeaders, detailRows) + "\n")
			sb.WriteString("\n")

			if hasVariables(rep.Flag.DefaultVariables) {
				sb.WriteString(s.Info.Bold(true).Render("  Default Variables") + "\n")
				renderVariablesLipGloss(&sb, s, rep.Flag.DefaultVariables, "    ")
				sb.WriteString("\n")
			}
		}

		// Variants
		if opts.ShowVariants {
			sb.WriteString(s.Info.Bold(true).Render("  Variants") + "\n")
			if len(rep.Flag.Overrides) == 0 {
				sb.WriteString("    No variants defined.\n")
				sb.WriteString("\n")
			} else {
				for _, ov := range rep.Flag.Overrides {
					sb.WriteString(fmt.Sprintf("    • %s (%s):\n", s.Text.Bold(true).Render(ov.Key), s.Muted.Render(ov.Name)))
					if ov.Description != "" {
						sb.WriteString(fmt.Sprintf("      Description: %s\n", ov.Description))
					}
					diffVars := GetDifferentVariables(ov.Variables, rep.Flag.DefaultVariables)
					if hasVariables(diffVars) {
						sb.WriteString("      Variables (Overrides):\n")
						renderVariablesLipGloss(&sb, s, diffVars, "        ")
					}
					sb.WriteString("\n")
				}
			}
		}

		// Environment configurations
		if opts.ShowEnvironments {
			sb.WriteString(s.Info.Bold(true).Render("  Environments") + "\n")
			if len(rep.Instances) == 0 {
				sb.WriteString("    No environments configured.\n")
				sb.WriteString("\n")
			} else {
				envHeaders := []string{"Environment", "Enabled", "Variant"}
				var envRows [][]string
				for _, inst := range rep.Instances {
					var selectedVariant string
					for _, target := range rep.Flag.Targets {
						if target.EnvironmentID == inst.EnvironmentID {
							if target.OverrideID != "" {
								variantName := target.OverrideID
								for _, ov := range rep.Flag.Overrides {
									if ov.Key == target.OverrideID || ov.ID == target.OverrideID {
										variantName = fmt.Sprintf("%s (%s)", ov.Key, ov.Name)
										break
									}
								}
								selectedVariant = variantName
							}
							break
						}
					}
					if selectedVariant == "" {
						selectedVariant = "-"
					}

					envRows = append(envRows, []string{
						inst.EnvironmentID,
						enabledStr(inst.Enabled),
						selectedVariant,
					})
				}
				sb.WriteString(RenderTable(envHeaders, envRows) + "\n")
				sb.WriteString("\n")

				hasAnyOverrides := false
				for _, inst := range rep.Instances {
					diffVars := GetDifferentVariables(inst.Variables, rep.Flag.DefaultVariables)
					if hasVariables(diffVars) {
						if !hasAnyOverrides {
							sb.WriteString(s.Muted.Bold(true).Render("    Variable Overrides by Environment:") + "\n")
							hasAnyOverrides = true
						}
						sb.WriteString(s.Text.Bold(true).Render("      "+inst.EnvironmentID) + ":\n")
						renderVariablesLipGloss(&sb, s, diffVars, "        ")
						sb.WriteString("\n")
					}
				}
				if hasAnyOverrides {
					sb.WriteString("\n")
				}
			}
		}

		if idx < len(reports)-1 {
			sb.WriteString(s.Muted.Render(strings.Repeat("═", 50)) + "\n")
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}

// renderVariablesLipGloss prints a core.Variables structure with LipGloss formatting to w.
func renderVariablesLipGloss(w io.Writer, s Styles, vars core.Variables, indent string) {
	for key, v := range vars.BoolVariables {
		renderVariableLipGloss(w, s, key, "boolean", fmt.Sprintf("%v", v.Value), v.Description, indent)
	}
	for key, v := range vars.IntVariables {
		renderVariableLipGloss(w, s, key, "integer", fmt.Sprintf("%v", v.Value), v.Description, indent)
	}
	for key, v := range vars.FloatVariables {
		renderVariableLipGloss(w, s, key, "float", fmt.Sprintf("%v", v.Value), v.Description, indent)
	}
	for key, v := range vars.StringVariables {
		renderVariableLipGloss(w, s, key, "string", fmt.Sprintf("%q", v.Value), v.Description, indent)
	}
	for key, v := range vars.JsonVariables {
		renderVariableLipGloss(w, s, key, "json", marshalJSON(v.Value), v.Description, indent)
	}
}

// renderVariableLipGloss prints a single variable with proper indentation for multiline JSON to w.
func renderVariableLipGloss(w io.Writer, s Styles, name, typ, value, description string, indent string) {
	fmt.Fprintf(w, "%s• %s (%s):\n", indent, s.Text.Bold(true).Render(name), s.Muted.Render(typ))
	if description != "" {
		fmt.Fprintf(w, "%s  Description: %s\n", indent, description)
	}
	if typ == "json" {
		fmt.Fprintf(w, "%s  Value:\n", indent)
		lines := strings.Split(value, "\n")
		for _, line := range lines {
			fmt.Fprintf(w, "%s    %s\n", indent, line)
		}
	} else {
		fmt.Fprintf(w, "%s  Value: %s\n", indent, value)
	}
}

// GetDifferentVariables returns a core.Variables containing only variables from `vars` that differ from `defaults`.
func GetDifferentVariables(vars core.Variables, defaults core.Variables) core.Variables {
	diff := core.Variables{
		BoolVariables:   make(core.VariableMap[bool]),
		IntVariables:    make(core.VariableMap[int]),
		FloatVariables:  make(core.VariableMap[float64]),
		StringVariables: make(core.VariableMap[string]),
		JsonVariables:   make(core.VariableMap[any]),
	}

	for key, v := range vars.BoolVariables {
		d, exists := defaults.BoolVariables[key]
		if !exists || v.Value != d.Value {
			diff.BoolVariables[key] = v
		}
	}
	for key, v := range vars.IntVariables {
		d, exists := defaults.IntVariables[key]
		if !exists || v.Value != d.Value {
			diff.IntVariables[key] = v
		}
	}
	for key, v := range vars.FloatVariables {
		d, exists := defaults.FloatVariables[key]
		if !exists || v.Value != d.Value {
			diff.FloatVariables[key] = v
		}
	}
	for key, v := range vars.StringVariables {
		d, exists := defaults.StringVariables[key]
		if !exists || v.Value != d.Value {
			diff.StringVariables[key] = v
		}
	}
	for key, v := range vars.JsonVariables {
		d, exists := defaults.JsonVariables[key]
		if !exists || !jsonEqual(v.Value, d.Value) {
			diff.JsonVariables[key] = v
		}
	}

	return diff
}

func jsonEqual(a, b any) bool {
	aS, okA := a.(string)
	bS, okB := b.(string)
	if okA && okB {
		return aS == bS
	}
	aBytes, errA := json.Marshal(a)
	bBytes, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(aBytes) == string(bBytes)
}

// RenderFlagComparisonsLipGlossStr renders a slice of core.FlagComparison as a styled table string.
func RenderFlagComparisonsLipGlossStr(comparisons []core.FlagComparison) (string, error) {
	s := AutoStyles()
	var sb strings.Builder

	headers := []string{"Flag Key", "Name", "Exists In", "Missing In"}
	var rows [][]string
	for _, comp := range comparisons {
		var existsStrs []string
		for _, p := range comp.ExistsIn {
			existsStrs = append(existsStrs, fmt.Sprintf("%s (%s)", p.Name, p.ID))
		}
		var missingStrs []string
		for _, p := range comp.MissingIn {
			missingStrs = append(missingStrs, fmt.Sprintf("%s (%s)", p.Name, p.ID))
		}

		rows = append(rows, []string{
			comp.Key,
			comp.Name,
			strings.Join(existsStrs, ", "),
			strings.Join(missingStrs, ", "),
		})
	}

	sb.WriteString(s.Title.Render("  ⚡ FEATURE FLAG COMPARISON MISMATCHES") + "\n")
	sb.WriteString(s.Muted.Render(fmt.Sprintf("  Found %d mismatching flags across projects.", len(comparisons))) + "\n\n")
	sb.WriteString(RenderTable(headers, rows) + "\n")
	return sb.String(), nil
}
