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
