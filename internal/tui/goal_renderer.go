package tui

import (
	"fmt"
	"sattchel/internal/tracker/core"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/exp/slices"
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

	parentVal := styles.Muted.Render("None")
	if parent != nil {
		parentVal = fmt.Sprintf("%s (%s)", styles.Text.Render(parent.Name), styles.Muted.Render(parent.ID))
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
				relVal = styles.Text.Render(string(ch.Parent.Relationship))
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
