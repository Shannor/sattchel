package driving

import (
	"fmt"
	"slices"
	"strings"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"
	"sattchel/pkg/set"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/spf13/cobra"
)

func listGoals(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		projectID     string
		statuses      []string
		impacts       []string
		efforts       []string
		relationships []string
		memberIDs     []string
		filterQuery   string
		flatMode      bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Goals",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := projectID
			if !cmd.Flags().Changed("projectId") {
				if lastProj := cfg.CurrentProjectID(); lastProj != "" {
					pid = lastProj
				}
			}

			var (
				goals   []core.Goal
				project *core.Project
				err     error
			)

			_ = loader.Run("Getting goals ...", func() {
				goals, err = service.GetGoals(cmd.Context(), pid)
				if err != nil {
					return
				}
				if pid != "" {
					project, err = service.GetProject(cmd.Context(), pid)
				}
			})
			if err != nil {
				return err
			}

			if len(goals) == 0 {
				p := pid
				if project != nil && project.Label != "" {
					p = project.Label
				}
				writer.Info(fmt.Sprintf("No goals found for project %s", p))
				return nil
			}

			// Build the active-filter set — only populated when flags were explicitly set.
			hasFilters := cmd.Flags().Changed("status") ||
				cmd.Flags().Changed("impact") ||
				cmd.Flags().Changed("effort") ||
				cmd.Flags().Changed("relationship") ||
				cmd.Flags().Changed("member") ||
				cmd.Flags().Changed("filter")

			statusSet := set.NewFrom(statuses)
			impactSet := set.NewFrom(impacts)
			effortSet := set.NewFrom(efforts)
			relationshipSet := set.NewFrom(relationships)
			memberSet := set.NewFrom(memberIDs)

			matchesFilter := func(g *core.Goal) bool {
				if !hasFilters {
					return true
				}
				if cmd.Flags().Changed("filter") && filterQuery != "" {
					if !g.MatchesQuery(filterQuery) {
						return false
					}
				}
				if cmd.Flags().Changed("status") && !statusSet.Contains(string(g.Status)) {
					return false
				}
				if cmd.Flags().Changed("impact") && !impactSet.Contains(string(g.Impact)) {
					return false
				}
				if cmd.Flags().Changed("effort") && !effortSet.Contains(string(g.Effort)) {
					return false
				}
				if cmd.Flags().Changed("relationship") {
					rel := ""
					if g.Parent != nil && g.Parent.TargetID != "" {
						rel = string(g.Parent.Relationship)
						if rel == "" {
							rel = string(core.LinkOptional)
						}
					}
					if !relationshipSet.Contains(rel) {
						return false
					}
				}
				if cmd.Flags().Changed("member") {
					mid := ""
					if g.Member != nil {
						mid = g.Member.ID
					}
					if !memberSet.Contains(mid) {
						return false
					}
				}
				return true
			}

			currentGoalID := cfg.CurrentGoalID()
			styles := tui.AutoStyles()
			enumeratorStyle := lipgloss.NewStyle().Foreground(styles.Success.GetForeground()).MarginRight(1)
			rootStyle := lipgloss.NewStyle().Foreground(styles.Title.GetForeground())
			itemStyle := lipgloss.NewStyle().Foreground(styles.Text.GetForeground())

			headerTitle := "Goals"
			if project != nil && project.Label != "" {
				headerTitle = project.Label
			} else if pid != "" {
				headerTitle = pid
			}

			// --flat: skip the tree and just print matching goals as a flat list.
			if flatMode {
				t := tree.Root(styles.Title.Render(headerTitle))
				for i := range goals {
					g := &goals[i]
					if !matchesFilter(g) {
						continue
					}
					t = t.Child(renderGoalLine(g, currentGoalID, styles, false))
				}
				fmt.Fprintln(cmd.OutOrStdout(), t.
					Enumerator(tree.RoundedEnumerator).
					EnumeratorStyle(enumeratorStyle).
					RootStyle(rootStyle).
					ItemStyle(itemStyle).
					String())
				return nil
			}

			// Default: tree view. Non-matching goals are dimmed but still shown
			// so the structure stays readable.
			t := tree.Root(styles.Title.Render(headerTitle))
			roots := buildGoalTree(goals)
			for _, root := range roots {
				t = t.Child(renderGoalTreeIterative(root, currentGoalID, styles, hasFilters, matchesFilter))
			}

			fmt.Fprintln(cmd.OutOrStdout(), t.
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(enumeratorStyle).
				RootStyle(rootStyle).
				ItemStyle(itemStyle).
				String())
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	cmd.Flags().StringVarP(&filterQuery, "filter", "f", "", "Filter goals by name/label query")
	cmd.Flags().StringSliceVarP(&statuses, "status", "s", nil, "Filter by status (draft, open, in-progress, completed, cancelled). Comma-separated or repeated.")
	cmd.Flags().StringSliceVarP(&impacts, "impact", "i", nil, "Filter by impact (low, medium, high). Comma-separated or repeated.")
	cmd.Flags().StringSliceVarP(&efforts, "effort", "e", nil, "Filter by effort (low, medium, high). Comma-separated or repeated.")
	cmd.Flags().StringSliceVarP(&relationships, "relationship", "r", nil, "Filter by link relationship (required, optional). Comma-separated or repeated.")
	cmd.Flags().StringSliceVarP(&memberIDs, "member", "m", nil, "Filter by member ID. Comma-separated or repeated.")
	cmd.Flags().BoolVar(&flatMode, "flat", false, "Show only matching goals as a flat list instead of the full tree")

	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"draft", "open", "in-progress", "completed", "cancelled"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("relationship", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{string(core.LinkOptional), string(core.LinkRequired)}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("impact", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{string(core.LowImpact), string(core.MediumImpact), string(core.HighImpact)}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("effort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{string(core.LowEffort), string(core.MediumEffort), string(core.HighEffort)}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("member", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getMemberCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

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

// renderGoalLine builds the styled single-line string for one goal card.
// dimmed=true renders the goal in muted colours for the tree-filter view.
func renderGoalLine(g *core.Goal, currentGoalID string, styles tui.Styles, dimmed bool) string {
	var nameText string
	if g.ID == currentGoalID {
		nameText = styles.Success.Bold(true).Render("★ " + g.Name)
	} else if dimmed {
		nameText = styles.Muted.Render(g.Name)
	} else {
		nameText = styles.Text.Bold(true).Render(g.Name)
	}

	if dimmed {
		return nameText
	}

	var details []string
	// Status
	if g.Status != "" {
		var statusStyle lipgloss.Style
		switch g.Status {
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
		details = append(details, statusStyle.Render(string(g.Status)))
	}

	if g.IsRoot() {
		details = append(details, styles.Title.Bold(true).Render("root"))
	} else if g.Parent != nil && g.Parent.TargetID != "" {
		rel := g.Parent.Relationship
		if rel == "" {
			rel = core.LinkOptional
		}
		var relStyle lipgloss.Style
		if rel == core.LinkRequired {
			relStyle = styles.Warning.Bold(true)
		} else {
			relStyle = styles.Muted
		}
		details = append(details, fmt.Sprintf("%s", relStyle.Render(string(rel))))
	}

	if g.Impact != "" && g.Impact != core.UnknownImpact {
		var impactStyle lipgloss.Style
		switch g.Impact {
		case core.HighImpact:
			impactStyle = styles.Success.Bold(true)
		case core.MediumImpact:
			impactStyle = styles.Info
		case core.LowImpact:
			impactStyle = styles.Muted
		default:
			impactStyle = styles.Warning
		}
		details = append(details, fmt.Sprintf("impact: %s", impactStyle.Render(string(g.Impact))))
	}

	// Effort
	if g.Effort != "" && g.Effort != core.UnknownEffort {
		var effortStyle lipgloss.Style
		switch g.Effort {
		case core.LowEffort:
			effortStyle = styles.Success
		case core.MediumEffort:
			effortStyle = styles.Info
		case core.HighEffort:
			effortStyle = styles.Error
		default:
			effortStyle = styles.Warning
		}
		details = append(details, fmt.Sprintf("effort: %s", effortStyle.Render(string(g.Effort))))
	}

	// Member
	if g.Member != nil && g.Member.Name != "" {
		details = append(details, styles.Info.Render("@"+g.Member.Name))
	}

	if len(details) > 0 {
		return fmt.Sprintf("%s \u2014 %s", nameText, strings.Join(details, styles.Muted.Render(" \u2022 ")))
	}
	return nameText
}

func renderGoalTreeIterative(root *GoalNode, currentGoalID string, styles tui.Styles, hasFilters bool, matchesFilter func(*core.Goal) bool) *tree.Tree {
	stack1 := []*GoalNode{root}
	var stack2 []*GoalNode

	for len(stack1) > 0 {
		curr := stack1[len(stack1)-1]
		stack1 = stack1[:len(stack1)-1]

		stack2 = append(stack2, curr)

		for _, child := range curr.Children {
			stack1 = append(stack1, child)
		}
	}

	nodeTrees := make(map[string]*tree.Tree)

	for len(stack2) > 0 {
		curr := stack2[len(stack2)-1]
		stack2 = stack2[:len(stack2)-1]

		dimmed := hasFilters && !matchesFilter(curr.Goal)
		title := renderGoalLine(curr.Goal, currentGoalID, styles, dimmed)

		t := tree.Root(title)

		for _, child := range curr.Children {
			childTree, ok := nodeTrees[child.Goal.ID]
			if ok {
				t = t.Child(childTree)
			}
		}

		nodeTrees[curr.Goal.ID] = t
	}

	return nodeTrees[root.Goal.ID]
}
