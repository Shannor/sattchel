package tui

import (
	"fmt"
	"strings"

	"sattchel/internal/tracker/core"

	"golang.org/x/exp/slices"
)

// RenderProjectDetails formats a Project, its members, and its goal statistics into a beautiful terminal display.
func RenderProjectDetails(project *core.Project, goals []core.Goal) string {
	styles := AutoStyles()
	var sb strings.Builder

	// Title Block
	sb.WriteString("\n")
	sb.WriteString(styles.Title.Render(fmt.Sprintf(" PROJECT DETAILS: %s ", strings.ToUpper(project.Label))) + "\n\n")

	// Overview Section
	sb.WriteString(styles.Info.Bold(true).Render("  OVERVIEW") + "\n")
	descVal := project.Description
	if descVal == "" {
		descVal = styles.Muted.Render("No description provided")
	} else {
		descVal = styles.Text.Render(descVal)
	}

	// Calculate goal statistics
	totalGoals := len(goals)
	completedGoals := 0
	inProgressGoals := 0
	for _, g := range goals {
		if g.Status == core.GoalCompleted {
			completedGoals++
		} else if g.Status == core.GoalInProgress {
			inProgressGoals++
		}
	}

	progressPct := 0.0
	if totalGoals > 0 {
		progressPct = (float64(completedGoals) / float64(totalGoals)) * 100.0
	}

	var progressStr string
	if totalGoals > 0 {
		progressBar := renderProgressBar(20, progressPct, styles)
		progressStr = fmt.Sprintf("%s  %.1f%% (%d/%d completed, %d in progress)",
			progressBar, progressPct, completedGoals, totalGoals, inProgressGoals)
	} else {
		progressStr = styles.Muted.Render("No goals defined yet")
	}

	overviewHeaders := []string{"Field", "Value"}
	overviewRows := [][]string{
		{"ID", styles.Text.Render(project.ID)},
		{"Name", styles.Text.Bold(true).Render(project.Label)},
		{"Description", descVal},
		{"Progress", progressStr},
	}
	sb.WriteString(RenderTable(overviewHeaders, overviewRows) + "\n\n")

	// Members Section
	sb.WriteString(styles.Info.Bold(true).Render("  MEMBERS WORKING ON PROJECT") + "\n")

	// Collect unique members from goals
	memberMap := make(map[string]*core.Member)
	for _, g := range goals {
		if g.Member != nil && g.Member.ID != "" {
			memberMap[g.Member.ID] = g.Member
		}
	}

	if len(memberMap) == 0 {
		sb.WriteString("    " + styles.Muted.Render("No members currently assigned to any goals.") + "\n\n")
	} else {
		// Sort members by name for deterministic rendering
		var members []*core.Member
		for _, m := range memberMap {
			members = append(members, m)
		}
		slices.SortFunc(members, func(a, b *core.Member) int {
			return strings.Compare(a.Name, b.Name)
		})

		for _, m := range members {
			emailStr := ""
			if m.Email != "" {
				emailStr = fmt.Sprintf(" (%s)", m.Email)
			}
			sb.WriteString(fmt.Sprintf("    %s %s%s\n",
				styles.Success.Render("•"),
				styles.Text.Bold(true).Render(m.Name),
				styles.Muted.Render(emailStr),
			))
		}
		sb.WriteString("\n")
	}

	// Goals Summary Section
	sb.WriteString(styles.Info.Bold(true).Render("  GOALS STATUS COMBINATION COUNTS") + "\n")
	if len(goals) == 0 {
		sb.WriteString("    " + styles.Muted.Render("No goals defined for this project.") + "\n\n")
	} else {
		// Calculate status counts
		var (
			completed  = 0
			inProgress = 0
			open       = 0
			cancelled  = 0
			draft      = 0
		)
		for _, g := range goals {
			switch g.Status {
			case core.GoalCompleted:
				completed++
			case core.GoalInProgress:
				inProgress++
			case core.GoalOpen:
				open++
			case core.GoalCancelled:
				cancelled++
			case core.GoalDraft:
				draft++
			}
		}

		statusHeaders := []string{"Status", "Count"}
		statusRows := [][]string{
			{styles.Success.Render("Completed"), fmt.Sprintf("%d", completed)},
			{styles.Info.Render("In Progress"), fmt.Sprintf("%d", inProgress)},
			{styles.Info.Render("Open"), fmt.Sprintf("%d", open)},
			{styles.Muted.Render("Cancelled"), fmt.Sprintf("%d", cancelled)},
			{styles.Warning.Render("Draft"), fmt.Sprintf("%d", draft)},
		}
		sb.WriteString(RenderTable(statusHeaders, statusRows) + "\n\n")

		// Calculate Triage category counts (similar to how triage command works)
		var (
			doItNow    = 0
			honestWork = 0
			snacking   = 0
			why        = 0
			missing    = 0
		)
		for _, g := range goals {
			if g.IsDoItNow() {
				doItNow++
			} else if g.IsHonestWork() {
				honestWork++
			} else if g.IsSnacking() {
				snacking++
			} else if g.IsWhy() {
				why++
			}

			mFields := getMissingFields(&g)
			if len(mFields) > 0 {
				missing++
			}
		}

		sb.WriteString(styles.Info.Bold(true).Render("  PRIORITY TRIAGE CATEGORIES") + "\n")
		triageHeaders := []string{"Triage Category (Impact/Effort)", "Count"}
		triageRows := [][]string{
			{styles.Success.Render("Do It Now (High Impact, Low Effort)"), fmt.Sprintf("%d", doItNow)},
			{styles.Info.Render("Honest Work (High Impact, High Effort)"), fmt.Sprintf("%d", honestWork)},
			{styles.Success.Render("Snacking (Low Impact, Low Effort)"), fmt.Sprintf("%d", snacking)},
			{styles.Warning.Render("Why? (Low Impact, High Effort)"), fmt.Sprintf("%d", why)},
			{styles.Error.Render("Missing Details (impact, effort, or assignee)"), fmt.Sprintf("%d", missing)},
		}
		sb.WriteString(RenderTable(triageHeaders, triageRows) + "\n\n")
	}

	return sb.String()
}

func renderProgressBar(width int, percentage float64, styles Styles) string {
	if percentage < 0 {
		percentage = 0
	} else if percentage > 100 {
		percentage = 100
	}

	filledLength := int(float64(width) * (percentage / 100.0))
	emptyLength := width - filledLength

	filledBar := strings.Repeat("█", filledLength)
	emptyBar := strings.Repeat("░", emptyLength)

	return styles.Success.Render(filledBar) + styles.Muted.Render(emptyBar)
}

func getMissingFields(g *core.Goal) []string {
	var missing []string
	if g.Member == nil || g.Member.ID == "" {
		missing = append(missing, "member")
	}
	if g.Impact == core.UnknownImpact || g.Impact == "" {
		missing = append(missing, "impact")
	}
	if g.Effort == core.UnknownEffort || g.Effort == "" {
		missing = append(missing, "effort")
	}
	return missing
}
