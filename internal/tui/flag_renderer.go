package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"sattchel/internal/optimizely/core"
)

// ProjectFlagReport represents feature flag definition and instances for a project.
type ProjectFlagReport struct {
	Project   core.Project
	Flag      *core.FeatureFlagDefinition
	Instances []core.FeatureFlagInstance
}

// ReportOptions controls which parts of the report are rendered.
type ReportOptions struct {
	ShowDetails      bool
	ShowVariants     bool
	ShowEnvironments bool
	ShowVariables    bool
}

func marshalJSON(v any) string {
	// If it's already a Go map/slice/etc, marshal directly.
	if _, ok := v.(map[string]any); ok {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
	// If it's a string, try to parse it as JSON and re-format.
	if s, ok := v.(string); ok {
		var parsed any
		if err := json.Unmarshal([]byte(s), &parsed); err == nil {
			b, err := json.MarshalIndent(parsed, "", "  ")
			if err == nil {
				return string(b)
			}
		}
		// Not valid JSON, just return as-is.
		return s
	}
	// Fallback for other types.
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
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
		return "Yes"
	}
	return "No"
}

// RenderMultiProjectFlagLipGlossStr renders multiple ProjectFlagReports using LipGloss styling and tables, returning the formatted string.
func RenderMultiProjectFlagLipGlossStr(reports []ProjectFlagReport, opts ReportOptions) (string, error) {
	if len(reports) == 0 {
		return "No data available.\n", nil
	}

	var sb strings.Builder
	s := AutoStyles()

	firstReport := reports[0]
	sb.WriteString(s.Title.Render(firstReport.Flag.Name) + "\n")
	sb.WriteString("\n")

	if firstReport.Flag.Description != "" {
		sb.WriteString(s.Text.Render("  "+firstReport.Flag.Description) + "\n")
		sb.WriteString("\n")
	}

	for idx, rep := range reports {
		// Project Header
		sb.WriteString(s.Info.Bold(true).Render("  "+strings.ToUpper(rep.Project.Name)) + "\n")
		sb.WriteString("\n")

		// Details
		if opts.ShowDetails {
			sb.WriteString(s.Info.Bold(true).Render("  Details") + "\n")
			status := "Active"
			if rep.Flag.Archived {
				status = "Archived"
			}

			createdStr := "-"
			if rep.Flag.CreatedAt != nil {
				createdStr = rep.Flag.CreatedAt.Format("2006-01-02")
				if rep.Flag.CreatedBy != nil {
					createdStr = fmt.Sprintf("%s (%s)", createdStr, *rep.Flag.CreatedBy)
				}
			}

			sourceURL := "-"
			if rep.Flag.SourceURL != "" {
				sourceURL = rep.Flag.SourceURL
				if strings.HasPrefix(sourceURL, "/projects/") {
					sourceURL = "https://app.optimizely.com/v2" + sourceURL
				} else if strings.HasPrefix(sourceURL, "/") {
					sourceURL = "https://app.optimizely.com" + sourceURL
				}
			}

			detailHeaders := []string{"Field", "Value"}
			detailRows := [][]string{
				{"Name", rep.Flag.Name},
				{"Key", rep.Flag.Key},
				{"ID", rep.Flag.ID},
				{"Status", status},
				{"Created", createdStr},
				{"Source", sourceURL},
			}
			sb.WriteString(RenderTable(detailHeaders, detailRows) + "\n")
			sb.WriteString("\n")
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
				if opts.ShowVariables {
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
				}
				if hasAnyOverrides {
					sb.WriteString("\n")
				}
			}
		}

		// Default Variables
		if opts.ShowDetails && opts.ShowVariables && hasVariables(rep.Flag.DefaultVariables) {
			sb.WriteString(s.Info.Bold(true).Render("  Default Variables") + "\n")
			renderVariablesLipGloss(&sb, s, rep.Flag.DefaultVariables, "    ")
			sb.WriteString("\n")
		}

		// Variants
		if opts.ShowVariants {
			sb.WriteString(s.Info.Bold(true).Render("  Variants") + "\n")
			if len(rep.Flag.Overrides) == 0 {
				sb.WriteString("    No variants defined.\n\n")
			} else {
				for _, ov := range rep.Flag.Overrides {
					sb.WriteString(fmt.Sprintf("    • %s", s.Text.Bold(true).Render(ov.Key)))
					if ov.Name != "" && ov.Name != ov.Key {
						sb.WriteString(fmt.Sprintf(" (%s)", s.Muted.Render(ov.Name)))
					}
					sb.WriteString("\n")
					if ov.Description != "" {
						sb.WriteString(fmt.Sprintf("      %s\n", s.Muted.Render(ov.Description)))
					}
					diffVars := GetDifferentVariables(ov.Variables, rep.Flag.DefaultVariables)
					if opts.ShowVariables {
						if hasVariables(diffVars) {
							sb.WriteString("      Variable Overrides:\n")
							renderVariablesLipGloss(&sb, s, diffVars, "        ")
						} else if hasVariables(rep.Flag.DefaultVariables) {
							sb.WriteString(fmt.Sprintf("      %s\n", s.Muted.Render("Same as Default")))
						}
					}
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
	var rows [][]string
	hasAnyDescription := false

	for key, v := range vars.BoolVariables {
		if v.Description != "" {
			hasAnyDescription = true
		}
		rows = append(rows, []string{key, "boolean", fmt.Sprintf("%v", v.Value), v.Description})
	}
	for key, v := range vars.IntVariables {
		if v.Description != "" {
			hasAnyDescription = true
		}
		rows = append(rows, []string{key, "integer", fmt.Sprintf("%v", v.Value), v.Description})
	}
	for key, v := range vars.FloatVariables {
		if v.Description != "" {
			hasAnyDescription = true
		}
		rows = append(rows, []string{key, "float", fmt.Sprintf("%v", v.Value), v.Description})
	}
	for key, v := range vars.StringVariables {
		if v.Description != "" {
			hasAnyDescription = true
		}
		rows = append(rows, []string{key, "string", fmt.Sprintf("%q", v.Value), v.Description})
	}
	for key, v := range vars.JsonVariables {
		if v.Description != "" {
			hasAnyDescription = true
		}
		rows = append(rows, []string{key, "json", marshalJSON(v.Value), v.Description})
	}

	if len(rows) == 0 {
		return
	}

	// Sort rows by variable name so that the output is deterministic and easy to read
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})

	var headers []string
	if hasAnyDescription {
		headers = []string{"Variable", "Type", "Value", "Description"}
	} else {
		headers = []string{"Variable", "Type", "Value"}
		for i := range rows {
			rows[i] = rows[i][:3]
		}
	}

	tableStr := RenderTable(headers, rows)

	// Apply indent to each line of the table string
	lines := strings.Split(tableStr, "\n")
	for i, line := range lines {
		if line != "" || i < len(lines)-1 {
			fmt.Fprintf(w, "%s%s\n", indent, line)
		}
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
