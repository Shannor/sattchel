package tui

import (
	"encoding/json"
	"fmt"
	"os"

	"sattchel/internal/optimizely/core"

	"charm.land/glamour/v2"
	"github.com/charmbracelet/x/term"
	"github.com/nao1215/markdown"
)

// RenderFlagGlamour renders the flag markdown output directly to stdout
// using Glamour's dark style, auto-detected terminal width.
func RenderFlagGlamour(flag *core.FeatureFlagDefinition, instances []core.FeatureFlagInstance) error {
	mdStr := buildFlagMarkdown(flag, instances)

	width, _, err := term.GetSize(uintptr(os.Stdout.Fd()))
	if err != nil {
		width = 80
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return fmt.Errorf("failed to create glamour renderer: %w", err)
	}
	defer r.Close()

	out, err := r.Render(mdStr)
	if err != nil {
		return fmt.Errorf("failed to render markdown: %w", err)
	}

	fmt.Print(out)
	return nil
}

// buildFlagMarkdown composes a markdown string from a feature flag and its instances.
func buildFlagMarkdown(flag *core.FeatureFlagDefinition, instances []core.FeatureFlagInstance) string {
	m := markdown.NewMarkdown(nil)

	// Header
	status := "✅ Active"
	if flag.Archived {
		status = "🗄️ Archived"
	}
	m.H1(flag.Name)
	m.LF()
	m.PlainText(fmt.Sprintf("*`%s`* — %s", flag.Key, status))
	m.LF()
	m.LF()

	// Description
	if flag.Description != "" {
		m.Blockquote(flag.Description)
		m.LF()
		m.LF()
	}

	// Details table
	m.H2("📋 Details")
	m.LF()
	m.Table(markdown.TableSet{
		Header: []string{"Field", "Value"},
		Rows: [][]string{
			{"ID", fmt.Sprintf("`%s`", flag.ID)},
			{"Archived", archivedStr(flag.Archived)},
		},
	})
	if flag.CreatedAt != nil {
		m.LF()
		m.PlainText(fmt.Sprintf("| Created | %s |", flag.CreatedAt.Format("2006-01-02")))
	}
	if flag.CreatedBy != nil {
		m.LF()
		m.PlainText(fmt.Sprintf("| Created By | %s |", *flag.CreatedBy))
	}
	m.LF()

	// Default variables
	if hasVariables(flag.DefaultVariables) {
		m.H2("⚙️ Default Variables")
		m.LF()
		renderVariables(m, flag.DefaultVariables)
		m.LF()
	}

	// Environment overrides
	if len(instances) > 0 {
		m.H2("🌍 Environment Overrides")
		m.LF()
		for i, inst := range instances {
			m.H3(inst.EnvironmentID)
			m.LF()
			m.Table(markdown.TableSet{
				Header: []string{"Field", "Value"},
				Rows: [][]string{
					{"Enabled", enabledStr(inst.Enabled)},
					{"Archived", archivedStr(inst.Archived)},
				},
			})
			m.LF()

			if hasVariables(inst.Variables) {
				m.PlainText("**Overrides:**")
				m.LF()
				renderVariables(m, inst.Variables)
				m.LF()
			}

			if i < len(instances)-1 {
				m.HorizontalRule()
				m.LF()
			}
		}
	}

	return m.String()
}

// renderVariables outputs each variable as a structured block.
func renderVariables(m *markdown.Markdown, vars core.Variables) {
	for key, v := range vars.BoolVariables {
		renderVariable(m, key, "boolean", fmt.Sprintf("%v", v.Value), v.Description)
	}
	for key, v := range vars.IntVariables {
		renderVariable(m, key, "integer", fmt.Sprintf("%v", v.Value), v.Description)
	}
	for key, v := range vars.FloatVariables {
		renderVariable(m, key, "float", fmt.Sprintf("%v", v.Value), v.Description)
	}
	for key, v := range vars.StringVariables {
		renderVariable(m, key, "string", v.Value, v.Description)
	}
	for key, v := range vars.JsonVariables {
		renderVariable(m, key, "json", marshalJSON(v.Value), v.Description)
	}
}

// renderVariable outputs a single variable as a key-value block.
func renderVariable(m *markdown.Markdown, name, typ, value, description string) {
	m.PlainText(fmt.Sprintf("**`%s`** *(`%s`)*", name, typ))
	m.LF()
	if description != "" {
		m.PlainText(description)
		m.LF()
	}
	if typ == "json" {
		m.PlainText("Value:")
		m.LF()
		m.CodeBlocks("json", value)
		m.LF()
	} else {
		m.PlainText(fmt.Sprintf("Value: `%s`", value))
		m.LF()
	}
	m.LF()
}

// marshalJSON formats a Go value as pretty-printed JSON with consistent 2-space indentation.
// If the value is a raw JSON string, it parses and re-formats it.
func marshalJSON(v any) string {
	// If it's already a Go map/slice/etc, marshal directly.
	if _, ok := v.(map[string]interface{}); ok {
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

func archivedStr(archived bool) string {
	if archived {
		return "🗄️ Yes"
	}
	return "❌ No"
}

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
}

// RenderMultiProjectFlagGlamour renders the multi-project flag markdown output directly to stdout
// using Glamour's dark style, auto-detected terminal width.
func RenderMultiProjectFlagGlamour(reports []ProjectFlagReport, opts ReportOptions) error {
	out, err := RenderMultiProjectFlagGlamourStr(reports, opts)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// RenderMultiProjectFlagGlamourStr renders the multi-project flag markdown output and returns it as a string.
func RenderMultiProjectFlagGlamourStr(reports []ProjectFlagReport, opts ReportOptions) (string, error) {
	mdStr := buildMultiProjectFlagMarkdown(reports, opts)

	width, _, err := term.GetSize(uintptr(os.Stdout.Fd()))
	if err != nil {
		width = 80
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create glamour renderer: %w", err)
	}
	defer r.Close()

	out, err := r.Render(mdStr)
	if err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	return out, nil
}

// buildMultiProjectFlagMarkdown composes a markdown string from a slice of multi-project reports.
func buildMultiProjectFlagMarkdown(reports []ProjectFlagReport, opts ReportOptions) string {
	m := markdown.NewMarkdown(nil)

	if len(reports) == 0 {
		m.H1("Feature Flag Report")
		m.PlainText("No data available.")
		return m.String()
	}

	firstReport := reports[0]
	m.H1(fmt.Sprintf("🚩 Feature Flag: %s", firstReport.Flag.Name))
	m.LF()
	m.PlainText(fmt.Sprintf("*Key:* `%s`", firstReport.Flag.Key))
	m.LF()
	m.LF()

	if firstReport.Flag.Description != "" {
		m.Blockquote(firstReport.Flag.Description)
		m.LF()
		m.LF()
	}

	for _, rep := range reports {
		m.H2(fmt.Sprintf("🏢 Project: %s (ID: %s)", rep.Project.Name, rep.Project.ID))
		m.LF()

		// Details section
		if opts.ShowDetails {
			m.H3("📋 Details")
			m.LF()
			status := "✅ Active"
			if rep.Flag.Archived {
				status = "🗄️ Archived"
			}
			m.Table(markdown.TableSet{
				Header: []string{"Field", "Value"},
				Rows: [][]string{
					{"Status", status},
					{"Flag ID", fmt.Sprintf("`%s`", rep.Flag.ID)},
					{"Archived", archivedStr(rep.Flag.Archived)},
				},
			})
			if rep.Flag.CreatedAt != nil {
				m.LF()
				m.PlainText(fmt.Sprintf("| Created | %s |", rep.Flag.CreatedAt.Format("2006-01-02")))
			}
			if rep.Flag.CreatedBy != nil {
				m.LF()
				m.PlainText(fmt.Sprintf("| Created By | %s |", *rep.Flag.CreatedBy))
			}
			m.LF()
			m.LF()

			if hasVariables(rep.Flag.DefaultVariables) {
				m.H3("⚙️ Default Variables")
				m.LF()
				renderVariables(m, rep.Flag.DefaultVariables)
				m.LF()
			}
		}

		// Variants (Variations) section
		if opts.ShowVariants {
			m.H3("🎨 Variants (Variations)")
			m.LF()
			if len(rep.Flag.Overrides) == 0 {
				m.PlainText("No variants defined.")
				m.LF()
			} else {
				for _, ov := range rep.Flag.Overrides {
					m.PlainText(fmt.Sprintf("- **`%s`** (%s)", ov.Key, ov.Name))
					m.LF()
					if ov.Description != "" {
						m.PlainText(fmt.Sprintf("  *Description:* %s", ov.Description))
						m.LF()
					}
					if hasVariables(ov.Variables) {
						m.PlainText("  *Variables:*")
						m.LF()
						for key, v := range ov.Variables.BoolVariables {
							m.PlainText(fmt.Sprintf("    - `%s` (boolean) = `%v`", key, v.Value))
							m.LF()
						}
						for key, v := range ov.Variables.IntVariables {
							m.PlainText(fmt.Sprintf("    - `%s` (integer) = `%v`", key, v.Value))
							m.LF()
						}
						for key, v := range ov.Variables.FloatVariables {
							m.PlainText(fmt.Sprintf("    - `%s` (float) = `%v`", key, v.Value))
							m.LF()
						}
						for key, v := range ov.Variables.StringVariables {
							m.PlainText(fmt.Sprintf("    - `%s` (string) = `\"%s\"`", key, v.Value))
							m.LF()
						}
						for key, v := range ov.Variables.JsonVariables {
							m.PlainText(fmt.Sprintf("    - `%s` (json) = `%s`", key, marshalJSON(v.Value)))
							m.LF()
						}
					}
					m.LF()
				}
			}
			m.LF()
		}

		// Environment setup section
		if opts.ShowEnvironments {
			m.H3("🌍 Environment Configurations")
			m.LF()
			if len(rep.Instances) == 0 {
				m.PlainText("No environments configured.")
				m.LF()
			} else {
				for _, inst := range rep.Instances {
					m.PlainText(fmt.Sprintf("#### **%s**", inst.EnvironmentID))
					m.LF()

					var selectedVariant string
					for _, target := range rep.Flag.Targets {
						if target.EnvironmentID == inst.EnvironmentID {
							if target.OverrideID != "" {
								variantName := target.OverrideID
								for _, ov := range rep.Flag.Overrides {
									if ov.Key == target.OverrideID || ov.ID == target.OverrideID {
										variantName = fmt.Sprintf("`%s` (%s)", ov.Key, ov.Name)
										break
									}
								}
								selectedVariant = variantName
							}
							break
						}
					}
					if selectedVariant == "" {
						selectedVariant = "*None (or Default)*"
					}

					m.PlainText(fmt.Sprintf("- **Enabled**: %s", enabledStr(inst.Enabled)))
					m.LF()
					m.PlainText(fmt.Sprintf("- **Selected Variant**: %s", selectedVariant))
					m.LF()

					if hasVariables(inst.Variables) {
						m.PlainText("- **Variable Overrides**:")
						m.LF()
						for key, v := range inst.Variables.BoolVariables {
							m.PlainText(fmt.Sprintf("  - `%s` (boolean) = `%v`", key, v.Value))
							m.LF()
						}
						for key, v := range inst.Variables.IntVariables {
							m.PlainText(fmt.Sprintf("  - `%s` (integer) = `%v`", key, v.Value))
							m.LF()
						}
						for key, v := range inst.Variables.FloatVariables {
							m.PlainText(fmt.Sprintf("  - `%s` (float) = `%v`", key, v.Value))
							m.LF()
						}
						for key, v := range inst.Variables.StringVariables {
							m.PlainText(fmt.Sprintf("  - `%s` (string) = `\"%s\"`", key, v.Value))
							m.LF()
						}
						for key, v := range inst.Variables.JsonVariables {
							m.PlainText(fmt.Sprintf("  - `%s` (json) = `%s`", key, marshalJSON(v.Value)))
							m.LF()
						}
					}
					m.LF()
				}
			}
			m.LF()
		}
	}

	return m.String()
}
