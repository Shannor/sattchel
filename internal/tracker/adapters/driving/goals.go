package driving

import (
	"fmt"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"slices"
	"strings"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/spf13/cobra"
)

func goals(service *core.Service, cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "goals [next]",
		Short:   "Manage goals",
		Aliases: []string{"g"},
		Long: `Manage goals.
   Examples:
     sattchel tracker goals create <name>
     sattchel tracker goals set
     sattchel tracker goals list
     `,
	}
	cmd.AddCommand(addGoal(service, cfg))
	cmd.AddCommand(setGoal(service, cfg))
	cmd.AddCommand(listGoals(service, cfg))
	return cmd
}

func addGoal(service *core.Service, cfg *Config) *cobra.Command {
	description := ""
	parentID := ""
	projectID := ""

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new goal",
		Long: `Add a new goal.
	If no key is provided, a list of available keys will be displayed.
   Examples:
     sattchel tracker goal add short
     sattchel tracker goal add "Long Title with Spaces"
     sattchel tracker goal add <name> -d="description" --parent=<parentId>
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("goal name is required")
			}

			pid := projectID
			if !cmd.Flags().Changed("projectId") {
				if lastProj := cfg.CurrentProjectID(); lastProj != "" {
					pid = lastProj
				}
			}

			parent := parentID
			if !cmd.Flags().Changed("parent") {
				if lastGoal := cfg.CurrentGoalID(); lastGoal != "" {
					parent = lastGoal
				}
			}

			options := core.GoalOptions{
				ParentID:    parent,
				Description: description,
			}
			goal, err := service.CreateGoal(cmd.Context(), pid, args[0], options)
			if err != nil {
				return err
			}
			fmt.Printf("Goal %s created successfully\n", goal.Name)
			_ = cfg.SetCurrentGoalID(goal.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the goal")
	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	cmd.Flags().StringVarP(&parentID, "parent", "", "", "Parent goal id of the goal. If not provided, the last parent will be used")
	return cmd
}

func setGoal(service *core.Service, cfg *Config) *cobra.Command {
	projectID := ""

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set Active Goal",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := projectID
			if !cmd.Flags().Changed("projectId") {
				if lastProj := cfg.CurrentProjectID(); lastProj != "" {
					pid = lastProj
				}
			}
			if pid == "" {
				return fmt.Errorf("no project selected")
			}

			var (
				goals []core.Goal
				err   error
			)

			if err := spinner.
				New().
				Title("Getting goals ...").
				Action(func() {
					goals, err = service.GetGoals(cmd.Context(), pid)
				}).Run(); err != nil {
				return err
			}

			if err != nil {
				return err
			}

			if len(goals) == 0 {
				return fmt.Errorf("no goals found for project %s", pid)
			}

			currentGoalID := cfg.CurrentGoalID()
			roots := buildGoalTree(goals)

			// Iterative DFS to build select options with tree indentation prefixes
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

			err = huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select Active Goal").
						Options(options...).
						Value(&selectedID),
				),
			).WithShowHelp(true).Run()
			if err != nil {
				return err
			}

			if err := cfg.SetCurrentGoalID(selectedID); err != nil {
				return fmt.Errorf("failed to save active goal: %w", err)
			}

			idx := slices.IndexFunc(goals, func(g core.Goal) bool { return g.ID == selectedID })
			g := goals[idx]

			fmt.Printf("Active goal set to: %s (%s)\n", g.Name, g.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	return cmd
}

func listGoals(service *core.Service, cfg *Config) *cobra.Command {
	projectID := ""
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
				goals []core.Goal
				err   error
			)

			if err := spinner.
				New().
				Title("Getting goals ...").
				Action(func() {
					goals, err = service.GetGoals(cmd.Context(), pid)
				}).Run(); err != nil {
				return err
			}

			if err != nil {
				return err
			}

			if len(goals) == 0 {
				fmt.Printf("No goals found for project %s\n", pid)
				return nil
			}

			currentGoalID := cfg.CurrentGoalID()
			styles := tui.AutoStyles()
			enumeratorStyle := lipgloss.NewStyle().Foreground(styles.Success.GetForeground()).MarginRight(1)
			rootStyle := lipgloss.NewStyle().Foreground(styles.Title.GetForeground())
			itemStyle := lipgloss.NewStyle().Foreground(styles.Text.GetForeground())

			t := tree.Root(styles.Title.Render("Goals"))
			roots := buildGoalTree(goals)
			for _, root := range roots {
				t = t.Child(renderGoalTreeIterative(root, currentGoalID, styles))
			}

			fmt.Println(t.
				Enumerator(tree.RoundedEnumerator).
				EnumeratorStyle(enumeratorStyle).
				RootStyle(rootStyle).
				ItemStyle(itemStyle).
				String())
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
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

func renderGoalTreeIterative(root *GoalNode, currentGoalID string, styles tui.Styles) *tree.Tree {
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

		currentMarker := ""
		if curr.Goal.ID == currentGoalID {
			currentMarker = " " + styles.Success.Render("(active)")
		}
		title := fmt.Sprintf("%s — %s%s", styles.Text.Bold(true).Render(curr.Goal.Name), styles.Muted.Render(curr.Goal.ID), currentMarker)

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
