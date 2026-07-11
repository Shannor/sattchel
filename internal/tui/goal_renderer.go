package tui

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"golang.org/x/exp/slices"
	"sattchel/internal/tracker/core"
	"strings"
)

type GoalNode struct {
	Goal     *core.Goal
	Children []*GoalNode
}

func buildGoalTree(goals []core.Goal) []*GoalNode {
	nodes := make(map[string]*GoalNode)
	for i := range goals {
		g := &goals[i]
		nodes[g.ID] = &GoalNode{Goal: g}
	}

	var roots []*GoalNode
	for _, node := range nodes {
		if node.Goal.Parent == nil || node.Goal.Parent.TargetID == "" {
			roots = append(roots, node)
		} else {
			parent, ok := nodes[node.Goal.Parent.TargetID]
			if ok {
				parent.Children = append(parent.Children, node)
			} else {
				roots = append(roots, node)
			}
		}
	}

	slices.SortFunc(roots, func(i, j *GoalNode) int {
		return strings.Compare(i.Goal.Name, j.Goal.Name)
	})

	var sortChildren func(n *GoalNode)
	sortChildren = func(n *GoalNode) {
		slices.SortFunc(n.Children, func(i, j *GoalNode) int {
			return strings.Compare(i.Goal.Name, j.Goal.Name)
		})
		for _, child := range n.Children {
			sortChildren(child)
		}
	}
	for _, root := range roots {
		sortChildren(root)
	}

	return roots
}

// ChooseGoal displays an interactive select form to choose a goal from a tree representation of goals.
func ChooseGoal(goals []core.Goal, title string, currentGoalID string, filterFn func(*core.Goal) (bool, bool), validateFn func(string) error) (string, error) {
	roots := buildGoalTree(goals)

	type stackElement struct {
		node   *GoalNode
		indent string
		isLast bool
	}

	var (
		selectedID string
		options    []huh.Option[string]
	)

	stack := make([]stackElement, 0)
	for i := len(roots) - 1; i >= 0; i-- {
		stack = append(stack, stackElement{
			node:   roots[i],
			indent: "",
			isLast: i == len(roots)-1,
		})
	}

	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		include := true
		traverseChildren := true
		if filterFn != nil {
			include, traverseChildren = filterFn(curr.node.Goal)
		}

		if !include && !traverseChildren {
			continue
		}

		if include {
			marker := ""
			if len(curr.indent) > 0 {
				if curr.isLast {
					marker = "└─ "
				} else {
					marker = "├─ "
				}
			}

			label := curr.indent + marker + curr.node.Goal.Name
			option := huh.NewOption(label, curr.node.Goal.ID)
			if curr.node.Goal.ID == currentGoalID {
				option = option.Selected(true)
				selectedID = curr.node.Goal.ID
			}
			options = append(options, option)
		}

		if traverseChildren {
			childIndent := curr.indent
			if len(curr.indent) > 0 {
				if curr.isLast {
					childIndent += "   "
				} else {
					childIndent += "│  "
				}
			} else {
				childIndent = "  "
			}

			for i := len(curr.node.Children) - 1; i >= 0; i-- {
				stack = append(stack, stackElement{
					node:   curr.node.Children[i],
					indent: childIndent,
					isLast: i == len(curr.node.Children)-1,
				})
			}
		}
	}

	if len(options) == 0 {
		return "", fmt.Errorf("no options available")
	}

	selectField := huh.NewSelect[string]().
		Title(title).
		Options(options...).
		Value(&selectedID)

	if validateFn != nil {
		selectField = selectField.Validate(validateFn)
	}

	err := huh.NewForm(
		huh.NewGroup(selectField),
	).WithShowHelp(true).Run()
	if err != nil {
		return "", err
	}

	return selectedID, nil
}

// RenderGoalDetails formats a Goal entity into a beautiful, styled string using Lipgloss.
func RenderGoalDetails(goal *core.Goal) string {
	styles := AutoStyles()
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(styles.Title.Render(" GOAL DETAILS ") + "\n\n")

	sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("ID:         "), styles.Text.Render(goal.ID)))
	sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Name:       "), styles.Text.Bold(true).Render(goal.Name)))

	var statusStyle lipgloss.Style
	switch goal.Status {
	case core.GoalCompleted:
		statusStyle = styles.Success.Bold(true)
	case core.GoalInProgress:
		statusStyle = styles.Info.Bold(true)
	case core.GoalCancelled:
		statusStyle = styles.Muted.Bold(true)
	case core.GoalOpen:
		statusStyle = styles.Info
	default:
		statusStyle = styles.Warning
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Status:     "), statusStyle.Render(string(goal.Status))))

	sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Project ID: "), styles.Text.Render(goal.ProjectID)))

	var impactStyle lipgloss.Style
	switch goal.Impact {
	case core.HighImpact:
		impactStyle = styles.Success.Bold(true)
	case core.MediumImpact:
		impactStyle = styles.Info
	case core.LowImpact:
		impactStyle = styles.Muted
	default:
		impactStyle = styles.Warning
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Impact:     "), impactStyle.Render(string(goal.Impact))))

	var effortStyle lipgloss.Style
	switch goal.Effort {
	case core.XSmallEffort, core.SmallEffort:
		effortStyle = styles.Success
	case core.MediumEffort:
		effortStyle = styles.Info
	case core.LargeEffort, core.XLargeEffort:
		effortStyle = styles.Error
	default:
		effortStyle = styles.Warning
	}
	sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Effort:     "), effortStyle.Render(string(goal.Effort))))

	if goal.Description != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Description:"), styles.Text.Render(goal.Description)))
	} else {
		sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Description:"), styles.Muted.Render("No description provided")))
	}

	if goal.Parent != nil && goal.Parent.TargetID != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Parent ID:  "), styles.Text.Render(goal.Parent.TargetID)))
	} else {
		sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Parent ID:  "), styles.Muted.Render("None")))
	}

	if len(goal.Children) > 0 {
		var childNames []string
		for _, ch := range goal.Children {
			childNames = append(childNames, fmt.Sprintf("%s (%s)", ch.Name, ch.ID))
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Children:   "), styles.Text.Render(strings.Join(childNames, ", "))))
	} else {
		sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Children:   "), styles.Muted.Render("None")))
	}

	if goal.Member != nil && goal.Member.ID != "" {
		sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Member ID:  "), styles.Text.Render(goal.Member.ID)))
	} else {
		sb.WriteString(fmt.Sprintf("%s %s\n", styles.Muted.Render("Member ID:  "), styles.Muted.Render("Unassigned")))
	}
	sb.WriteString("\n")

	return sb.String()
}
