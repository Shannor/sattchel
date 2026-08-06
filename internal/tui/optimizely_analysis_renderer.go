package tui

import (
	"fmt"
	"sattchel/internal/optimizely/core"
	"slices"
	"strings"
	"time"
)

func RenderOptimizelyCompareReport(report *core.FlagComparisonReport, baseProjectID, focusProjectID string) string {
	s := AutoStyles()
	var sb strings.Builder

	sb.WriteString(s.Title.Render("⚡ Optimizely Flag Comparison") + "\n")
	if baseProjectID != "" {
		sb.WriteString(s.Muted.Render("Base mode: show only flags missing from the selected base project") + "\n")
	}
	if focusProjectID != "" {
		sb.WriteString(s.Muted.Render("Focus mode: show only flags the selected focus project has that other projects are missing") + "\n")
	}
	sb.WriteString("\n")

	summaryRows := make([][]string, 0, len(report.Projects))
	for _, project := range report.Projects {
		summaryRows = append(summaryRows, []string{projectDisplay(project), fmt.Sprintf("%d", report.FlagCountByProject[project.ID])})
	}
	sb.WriteString(sectionTitle("Summary") + "\n")
	sb.WriteString(RenderTable([]string{"Project", "Flag Count"}, summaryRows) + "\n\n")

	metrics := [][]string{{"Shared flags", fmt.Sprintf("%d", len(report.SharedFlags))}, {"Missing flags", fmt.Sprintf("%d", len(report.MissingFlags))}}
	if baseProjectID != "" {
		metrics = append(metrics, []string{"Base project", projectDisplay(findProject(report.Projects, baseProjectID))})
	}
	if focusProjectID != "" {
		metrics = append(metrics, []string{"Focus project", projectDisplay(findProject(report.Projects, focusProjectID))})
	}
	sb.WriteString(RenderTable([]string{"Metric", "Value"}, metrics) + "\n\n")

	if len(report.MissingFlags) == 0 {
		sb.WriteString(s.Success.Render("All feature flags match across the selected projects.") + "\n")
		return sb.String()
	}

	missingRows := make([][]string, 0, len(report.MissingFlags))
	for _, entry := range report.MissingFlags {
		missingRows = append(missingRows, []string{
			entry.Flag.Key,
			fallback(entry.Flag.Name),
			projectDisplay(entry.SourceProject),
			joinProjects(entry.PresentIn),
			joinProjects(entry.MissingIn),
			formatCreatedAt(entry.Flag.CreatedAt),
		})
	}
	sb.WriteString(sectionTitle("Missing Flags") + "\n")
	sb.WriteString(RenderTable([]string{"Key", "Name", "Source", "Present In", "Missing In", "Created"}, missingRows) + "\n")
	return sb.String()
}

func RenderOptimizelyUniqueFlags(entries []core.UniqueFlagEntry) string {
	s := AutoStyles()
	var sb strings.Builder
	sb.WriteString(s.Title.Render("✨ Unique Feature Flags") + "\n\n")
	if len(entries) == 0 {
		sb.WriteString(s.Success.Render("No unique flags found.") + "\n")
		return sb.String()
	}

	for i, entry := range entries {
		sb.WriteString(fmt.Sprintf("  %s %s\n", s.Info.Render("•"), s.Text.Bold(true).Render(entry.Flag.Key)))
		sb.WriteString(fmt.Sprintf("    %-15s%s\n", s.Muted.Render("Name:"), fallback(entry.Flag.Name)))
		sb.WriteString(fmt.Sprintf("    %-15s%s\n", s.Muted.Render("Project:"), projectDisplay(entry.TargetProject)))
		sb.WriteString(fmt.Sprintf("    %-15s%s\n", s.Muted.Render("Created:"), formatCreatedAt(entry.Flag.CreatedAt)))
		
		sb.WriteString(fmt.Sprintf("    %-15s\n", s.Muted.Render("Absent From:")))
		for _, compProj := range entry.ComparedAgainst {
			sb.WriteString(fmt.Sprintf("      %s %s\n", s.Warning.Render("-"), projectDisplay(compProj)))
		}
		
		if i < len(entries)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func RenderOptimizelyDormantFlags(entries []core.DormantFlagEntry, projectIDs []string) string {
	s := AutoStyles()
	var sb strings.Builder
	sb.WriteString(s.Title.Render("🌙 Dormant Feature Flags") + "\n")
	sb.WriteString(s.Muted.Render("Disabled in every environment across the selected projects.") + "\n\n")
	sb.WriteString(sectionTitle("Projects Checked") + "\n")
	sb.WriteString(strings.Join(projectIDs, ", ") + "\n\n")
	if len(entries) == 0 {
		sb.WriteString(s.Success.Render("No dormant flags found.") + "\n")
		return sb.String()
	}

	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{
			entry.Flag.Key,
			fallback(entry.Flag.Name),
			joinProjects(entry.PresentIn),
			formatCreatedAt(entry.Flag.CreatedAt),
		})
	}
	sb.WriteString(RenderTable([]string{"Key", "Name", "Present In", "Created"}, rows) + "\n")
	return sb.String()
}

func RenderOptimizelyVariableDrift(entries []core.FlagVariableDrift) string {
	s := AutoStyles()
	var sb strings.Builder
	sb.WriteString(s.Title.Render("🧬 Variable Definition Drift") + "\n")
	sb.WriteString(s.Muted.Render("Flags that exist in multiple projects but disagree on variable shape or defaults.") + "\n\n")
	if len(entries) == 0 {
		sb.WriteString(s.Success.Render("No variable definition drift found.") + "\n")
		return sb.String()
	}

	quickRows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		varKeys := make([]string, 0, len(entry.Variables))
		for _, variable := range entry.Variables {
			varKeys = append(varKeys, variable.Key)
		}
		quickRows = append(quickRows, []string{entry.FlagKey, fallback(entry.FlagName), joinProjects(entry.PresentIn), strings.Join(varKeys, ", ")})
	}
	sb.WriteString(sectionTitle("Quick Reference") + "\n")
	sb.WriteString(RenderTable([]string{"Flag Key", "Name", "Present In", "Drifted Variables"}, quickRows) + "\n\n")

	for _, entry := range entries {
		sb.WriteString(sectionTitle(fmt.Sprintf("%s — %s", entry.FlagKey, fallback(entry.FlagName))) + "\n")
		for _, variable := range entry.Variables {
			rows := make([][]string, 0, len(variable.ValuesByProject))
			projectIDs := make([]string, 0, len(variable.ValuesByProject))
			for projectID := range variable.ValuesByProject {
				projectIDs = append(projectIDs, projectID)
			}
			slices.Sort(projectIDs)
			for _, projectID := range projectIDs {
				value := variable.ValuesByProject[projectID]
				exists := "no"
				if value.Exists {
					exists = "yes"
				}
				rows = append(rows, []string{projectID, exists, fallback(value.Type), fallback(value.DefaultValue), fallback(value.Description)})
			}
			sb.WriteString(AutoStyles().Info.Render("Variable: "+variable.Key) + "\n")
			sb.WriteString(RenderTable([]string{"Project", "Exists", "Type", "Default", "Description"}, rows) + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func RenderOptimizelyPromotionCandidates(entries []core.PromotionCandidate, targetProjectID string, againstProjectIDs []string) string {
	s := AutoStyles()
	var sb strings.Builder
	sb.WriteString(s.Title.Render("🚀 Promotion Candidates") + "\n")
	sb.WriteString(s.Muted.Render("Flags enabled only in lower environments that may need promotion.") + "\n\n")
	metaRows := [][]string{{"Target project", targetProjectID}, {"Compared against", strings.Join(againstProjectIDs, ", ")}, {"Candidate count", fmt.Sprintf("%d", len(entries))}}
	sb.WriteString(RenderTable([]string{"Metric", "Value"}, metaRows) + "\n\n")
	if len(entries) == 0 {
		sb.WriteString(s.Success.Render("No promotion candidates found.") + "\n")
		return sb.String()
	}

	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{
			entry.Flag.Key,
			fallback(entry.Flag.Name),
			string(entry.Reason),
			fallback(strings.Join(entry.EnabledEnvironments, ", ")),
			joinProjects(entry.PresentIn),
		})
	}
	sb.WriteString(RenderTable([]string{"Key", "Name", "Reason", "Enabled Envs", "Present In"}, rows) + "\n")
	return sb.String()
}

func RenderOptimizelySyncPlan(plan core.FlagSyncPlan, dryRun bool, result *core.FlagSyncResult) string {
	s := AutoStyles()
	var sb strings.Builder
	title := "🔄 Optimizely Flag Sync"
	if dryRun {
		title += " (dry run)"
	}
	sb.WriteString(s.Title.Render(title) + "\n\n")

	source := plan.SourceProjectID
	if plan.UnionSource {
		source = "all configured targets (union mode)"
	}
	metaRows := [][]string{{"Source", source}, {"Projects in scope", joinProjects(plan.Projects)}, {"Flag creations", fmt.Sprintf("%d", totalFlagCreations(plan))}, {"Variable additions", fmt.Sprintf("%d", totalVariableAdditions(plan))}}
	sb.WriteString(RenderTable([]string{"Metric", "Value"}, metaRows) + "\n\n")

	if len(plan.TargetMissing) == 0 && len(plan.TargetVariableUpdates) == 0 {
		sb.WriteString(s.Success.Render("No changes needed.") + "\n")
		if result != nil {
			sb.WriteString("\n" + renderSyncResultTable(*result))
		}
		return sb.String()
	}

	if len(plan.TargetMissing) > 0 {
		rows := make([][]string, 0)
		for _, target := range plan.TargetMissing {
			for _, flag := range target.Flags {
				rows = append(rows, []string{projectDisplay(target.Project), flag.Key, fallback(flag.Name), fmt.Sprintf("%d", len(flag.DefaultVariables.Definitions()))})
			}
		}
		sb.WriteString(sectionTitle("Flags to Create") + "\n")
		sb.WriteString(RenderTable([]string{"Project", "Flag Key", "Name", "Variables"}, rows) + "\n\n")
	}

	if len(plan.TargetVariableUpdates) > 0 {
		rows := make([][]string, 0)
		for _, target := range plan.TargetVariableUpdates {
			for _, update := range target.Updates {
				keys := make([]string, 0, len(update.MissingVariables))
				for key := range update.MissingVariables {
					keys = append(keys, key)
				}
				slices.Sort(keys)
				rows = append(rows, []string{projectDisplay(target.Project), update.FlagKey, strings.Join(keys, ", ")})
			}
		}
		sb.WriteString(sectionTitle("Variable Definitions to Add") + "\n")
		sb.WriteString(RenderTable([]string{"Project", "Flag Key", "Variables"}, rows) + "\n")
	}

	if result != nil {
		sb.WriteString("\n" + renderSyncResultTable(*result))
	}
	return sb.String()
}

func renderSyncResultTable(result core.FlagSyncResult) string {
	return sectionTitle("Apply Result") + "\n" + RenderTable(
		[]string{"Metric", "Value"},
		[][]string{{"Created flags", fmt.Sprintf("%d", result.CreatedFlags)}, {"Added variables", fmt.Sprintf("%d", result.AddedVariables)}, {"Touched projects", fmt.Sprintf("%d", result.TouchedProjects)}},
	) + "\n"
}

func totalFlagCreations(plan core.FlagSyncPlan) int {
	total := 0
	for _, target := range plan.TargetMissing {
		total += len(target.Flags)
	}
	return total
}

func totalVariableAdditions(plan core.FlagSyncPlan) int {
	total := 0
	for _, target := range plan.TargetVariableUpdates {
		for _, update := range target.Updates {
			total += len(update.MissingVariables)
		}
	}
	return total
}

func sectionTitle(title string) string {
	return AutoStyles().Title.Render(title)
}

func findProject(projects []core.Project, id string) core.Project {
	for _, project := range projects {
		if project.ID == id {
			return project
		}
	}
	return core.Project{ID: id, Name: id}
}

func projectDisplay(project core.Project) string {
	name := strings.TrimSpace(project.Name)
	if name == "" || name == project.ID {
		return project.ID
	}
	return fmt.Sprintf("%s (%s)", name, project.ID)
}

func joinProjects(projects []core.Project) string {
	if len(projects) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(projects))
	for _, project := range projects {
		parts = append(parts, projectDisplay(project))
	}
	return strings.Join(parts, ", ")
}

func formatCreatedAt(createdAt *time.Time) string {
	if createdAt == nil {
		return "—"
	}
	return createdAt.UTC().Format(time.RFC3339)
}

func fallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}
