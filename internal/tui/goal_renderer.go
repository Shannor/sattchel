package tui

import (
	"fmt"
	"sattchel/internal/tracker/core"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/exp/slices"
)

type GoalNode struct {
	Goal     *core.Goal
	Children []*GoalNode
}

// ChooseGoal displays an interactive filterable select list to choose a goal.
func ChooseGoal(goals []core.Goal, title string, currentGoalID string, filterFn func(*core.Goal) bool, validateFn func(string) error) (string, error) {
	sortedGoals := slices.Clone(goals)
	slices.SortFunc(sortedGoals, func(i, j core.Goal) int {
		return strings.Compare(strings.ToLower(i.Name), strings.ToLower(j.Name))
	})

	var listOpts []ListOption
	for i := range sortedGoals {
		g := &sortedGoals[i]
		if filterFn != nil && !filterFn(g) {
			continue
		}

		var descParts []string
		if g.ID == currentGoalID {
			descParts = append(descParts, "★ active")
		}
		if g.IsRoot() {
			descParts = append(descParts, "root")
		} else if g.HasParent() {
			rel := core.LinkOptional
			if g.Parent != nil && g.Parent.Relationship != "" {
				rel = g.Parent.Relationship
			}
			descParts = append(descParts, fmt.Sprintf("link: %s", rel))
		}
		if g.Status != "" {
			descParts = append(descParts, string(g.Status))
		}
		if g.Member != nil && g.Member.Name != "" {
			descParts = append(descParts, "@"+g.Member.Name)
		}
		descParts = append(descParts, fmt.Sprintf("id: %s", g.ID))

		listOpts = append(listOpts, ListOption{
			TitleStr:       g.Name,
			DescriptionStr: strings.Join(descParts, " • "),
			ValueStr:       g.ID,
		})
	}

	if len(listOpts) == 0 {
		return "", fmt.Errorf("no options available")
	}

	selected, err := Choose(title, listOpts)
	if err != nil {
		return "", err
	}
	if selected == nil {
		return "", fmt.Errorf("no goal selected")
	}

	if validateFn != nil {
		if err := validateFn(selected.ValueStr); err != nil {
			return "", err
		}
	}

	return selected.ValueStr, nil
}

// ChooseProject displays an interactive filterable select list to choose a project.
func ChooseProject(projects []core.Project, title string, currentProjectID string) (string, error) {
	sortedProjects := slices.Clone(projects)
	slices.SortFunc(sortedProjects, func(i, j core.Project) int {
		return strings.Compare(strings.ToLower(i.Label), strings.ToLower(j.Label))
	})

	var listOpts []ListOption
	for i := range sortedProjects {
		p := &sortedProjects[i]
		var descParts []string
		if p.ID == currentProjectID {
			descParts = append(descParts, "★ active")
		}
		if p.Description != "" {
			descParts = append(descParts, p.Description)
		}
		descParts = append(descParts, fmt.Sprintf("id: %s", p.ID))

		listOpts = append(listOpts, ListOption{
			TitleStr:       p.Label,
			DescriptionStr: strings.Join(descParts, " • "),
			ValueStr:       p.ID,
		})
	}

	if len(listOpts) == 0 {
		return "", fmt.Errorf("no options available")
	}

	selected, err := Choose(title, listOpts)
	if err != nil {
		return "", err
	}
	if selected == nil {
		return "", fmt.Errorf("no project selected")
	}

	return selected.ValueStr, nil
}

// ChooseMember displays an interactive filterable select list to choose a member.
func ChooseMember(members []core.Member, title string, includeUnassigned bool) (string, error) {
	sortedMembers := slices.Clone(members)
	slices.SortFunc(sortedMembers, func(i, j core.Member) int {
		return strings.Compare(strings.ToLower(i.Name), strings.ToLower(j.Name))
	})

	var listOpts []ListOption
	if includeUnassigned {
		listOpts = append(listOpts, ListOption{
			TitleStr:       "Unassigned",
			DescriptionStr: "Do not assign to a member",
			ValueStr:       "",
		})
	}
	for i := range sortedMembers {
		m := &sortedMembers[i]
		var descParts []string
		if m.Email != "" {
			descParts = append(descParts, m.Email)
		}
		descParts = append(descParts, fmt.Sprintf("id: %s", m.ID))

		listOpts = append(listOpts, ListOption{
			TitleStr:       m.Name,
			DescriptionStr: strings.Join(descParts, " • "),
			ValueStr:       m.ID,
		})
	}

	if len(listOpts) == 0 {
		return "", fmt.Errorf("no members available")
	}

	selected, err := Choose(title, listOpts)
	if err != nil {
		return "", err
	}
	if selected == nil {
		return "", fmt.Errorf("no member selected")
	}

	return selected.ValueStr, nil
}

// RenderGoalDetails formats a Goal entity into a beautiful, styled string using Lipgloss tables.
func RenderGoalDetails(goal *core.Goal, parent *core.Goal) string {
	styles := AutoStyles()
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(styles.Title.Render(" GOAL DETAILS ") + "\n\n")

	statusVal := getStatusStyle(goal.Status, styles).Render(string(goal.Status))
	impactVal := getImpactStyle(goal.Impact, styles).Render(string(goal.Impact))
	effortVal := getEffortStyle(goal.Effort, styles).Render(string(goal.Effort))

	descVal := goal.Description
	if descVal == "" {
		descVal = styles.Muted.Render("No description provided")
	} else {
		descVal = styles.Text.Render(descVal)
	}

	parentVal := styles.Muted.Render("None (root)")
	if parent != nil {
		parentVal = fmt.Sprintf("%s (%s)", styles.Text.Render(parent.Name), styles.Muted.Render(parent.ID))
		if goal.Parent != nil && goal.Parent.Relationship != "" {
			relStyle := styles.Muted
			if goal.Parent.Relationship == core.LinkRequired {
				relStyle = styles.Warning.Bold(true)
			}
			parentVal += fmt.Sprintf(" [%s]", relStyle.Render(string(goal.Parent.Relationship)))
		}
	}

	memberVal := styles.Muted.Render("Unassigned")
	if goal.Member != nil && goal.Member.ID != "" {
		memberVal = fmt.Sprintf("%s (%s)", styles.Text.Render(goal.Member.Name), styles.Muted.Render(goal.Member.ID))
	}

	detailHeaders := []string{"Field", "Value"}
	detailRows := [][]string{
		{"ID", styles.Text.Render(goal.ID)},
		{"Name", styles.Text.Bold(true).Render(goal.Name)},
		{"Status", statusVal},
		{"Project ID", styles.Text.Render(goal.ProjectID)},
		{"Impact", impactVal},
		{"Effort", effortVal},
		{"Description", descVal},
		{"Parent", parentVal},
		{"Member", memberVal},
	}

	sb.WriteString(RenderTable(detailHeaders, detailRows) + "\n\n")

	if len(goal.Children) > 0 {
		sb.WriteString(styles.Title.Render(" CHILDREN GOALS ") + "\n\n")
		slices.SortFunc(goal.Children, func(a, b core.Goal) int { return a.Compare(b) })
		childHeaders := []string{"ID", "Name", "Relationship", "Status", "Impact", "Effort", "Member"}
		var childRows [][]string
		for _, ch := range goal.Children {
			chStatusVal := getStatusStyle(ch.Status, styles).Render(string(ch.Status))
			chImpactVal := getImpactStyle(ch.Impact, styles).Render(string(ch.Impact))
			chEffortVal := getEffortStyle(ch.Effort, styles).Render(string(ch.Effort))
			member := styles.Text.Render("Unassigned")
			if ch.HasMember() {
				member = styles.Text.Render(ch.Member.Name)
			}
			relVal := styles.Text.Render("-")
			if ch.Parent != nil && ch.Parent.Relationship != "" {
				if ch.Parent.Relationship == core.LinkRequired {
					relVal = styles.Warning.Bold(true).Render(string(ch.Parent.Relationship))
				} else {
					relVal = styles.Muted.Render(string(ch.Parent.Relationship))
				}
			}
			childRows = append(childRows, []string{
				styles.Text.Render(ch.ID),
				styles.Text.Bold(true).Render(ch.Name),
				relVal,
				chStatusVal,
				chImpactVal,
				chEffortVal,
				member,
			})
		}
		sb.WriteString(RenderTable(childHeaders, childRows) + "\n")
	} else {
		sb.WriteString(styles.Muted.Render("  No children goals") + "\n\n")
	}

	return sb.String()
}

func getStatusStyle(status core.GoalStatus, styles Styles) lipgloss.Style {
	switch status {
	case core.GoalCompleted:
		return styles.Success.Bold(true)
	case core.GoalInProgress:
		return styles.Info.Bold(true)
	case core.GoalCancelled:
		return styles.Muted.Bold(true)
	case core.GoalOpen:
		return styles.Info
	default:
		return styles.Warning
	}
}

func getImpactStyle(impact core.Impact, styles Styles) lipgloss.Style {
	switch impact {
	case core.HighImpact:
		return styles.Success.Bold(true)
	case core.MediumImpact:
		return styles.Info
	case core.LowImpact:
		return styles.Muted
	default:
		return styles.Warning
	}
}

func getEffortStyle(effort core.Effort, styles Styles) lipgloss.Style {
	switch effort {
	case core.LowEffort:
		return styles.Success
	case core.MediumEffort:
		return styles.Info
	case core.HighEffort:
		return styles.Error
	default:
		return styles.Warning
	}
}
