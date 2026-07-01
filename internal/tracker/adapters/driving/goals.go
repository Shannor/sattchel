package driving

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tracker/visualizer"
	"sattchel/internal/tui"
	"slices"
	"strings"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/spf13/cobra"
)

func getProjectCompletions(service *core.Service) []string {
	projects, err := service.GetProjects(context.Background())
	if err != nil {
		return nil
	}
	var completions []string
	for _, p := range projects {
		completions = append(completions, fmt.Sprintf("%s\t%s", p.ID, p.Label))
	}
	return completions
}

func getGoalCompletions(service *core.Service, pid string) []string {
	if pid == "" {
		return nil
	}
	goals, err := service.GetGoals(context.Background(), pid)
	if err != nil {
		return nil
	}
	var completions []string
	for _, g := range goals {
		completions = append(completions, fmt.Sprintf("%s\t%s", g.ID, g.Name))
	}
	return completions
}

func getActiveProjectID(cmd *cobra.Command, cfg *Config, projectIDFlag string) string {
	if projectIDFlag != "" {
		return projectIDFlag
	}
	if cmd.Flags().Changed("projectId") {
		if pid, err := cmd.Flags().GetString("projectId"); err == nil && pid != "" {
			return pid
		}
	}
	if lastProj := cfg.CurrentProjectID(); lastProj != "" {
		return lastProj
	}
	return ""
}

func goals(service *core.Service, cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "goals [verb]",
		Short:   "Manage goals",
		Aliases: []string{"g"},
		Long: `Commands to manage goals.
   Examples:
     sattchel tracker goals add <name>
     sattchel tracker goals set
     sattchel tracker goals list
     sattchel tracker goals move <childId> <newParentId>
     sattchel tracker goals visualize
     `,
	}
	cmd.AddCommand(addGoal(service, cfg))
	cmd.AddCommand(setGoal(service, cfg))
	cmd.AddCommand(listGoals(service, cfg))
	cmd.AddCommand(moveGoal(service, cfg))
	cmd.AddCommand(visualizeGoals(service, cfg))
	return cmd
}

func addGoal(service *core.Service, cfg *Config) *cobra.Command {
	description := ""
	parentID := ""
	projectID := ""
	changeCurrent := false
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new goal",
		Long: `Add a new goal.
	Will create a new goal. If it's the root goal it will automatically get set as current'.
	For each goal after it will stay pointing at root unless you provide a parent or flag on creation to change it.
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
			if changeCurrent {
				_ = cfg.SetCurrentGoalID(goal.ID)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the goal")
	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	cmd.Flags().StringVarP(&parentID, "parent", "", "", "Parent goal id of the goal. If not provided, the last parent will be used")
	cmd.Flags().BoolVarP(&changeCurrent, "set", "s", false, "Set the newly created goal as current")

	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("parent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		pid := getActiveProjectID(cmd, cfg, projectID)
		return getGoalCompletions(service, pid), cobra.ShellCompDirectiveNoFileComp
	})

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
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
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
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
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

func buildSelectOptions(roots []*GoalNode, filterFn func(*core.Goal) (bool, bool), rootGoalID string) []huh.Option[string] {
	type stackElement struct {
		node   *GoalNode
		indent string
		isLast bool
	}

	var options []huh.Option[string]
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
			if curr.node.Goal.ID == rootGoalID {
				label += " (root - cannot move)"
			}
			option := huh.NewOption(label, curr.node.Goal.ID)
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
	return options
}

func getExcludedSubtreeIDs(roots []*GoalNode, childID string) map[string]bool {
	nodesMap := make(map[string]*GoalNode)
	var traverse func(*GoalNode)
	traverse = func(n *GoalNode) {
		nodesMap[n.Goal.ID] = n
		for _, child := range n.Children {
			traverse(child)
		}
	}
	for _, root := range roots {
		traverse(root)
	}

	excluded := make(map[string]bool)
	childNode, ok := nodesMap[childID]
	if !ok {
		excluded[childID] = true
		return excluded
	}

	stack := []*GoalNode{childNode}
	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		excluded[curr.Goal.ID] = true
		for _, child := range curr.Children {
			stack = append(stack, child)
		}
	}
	return excluded
}

func moveGoal(service *core.Service, cfg *Config) *cobra.Command {
	projectID := ""

	cmd := &cobra.Command{
		Use:     "move [childId] [newParentId]",
		Short:   "Move a goal to a new parent",
		Aliases: []string{"mv"},
		Long: `Move a goal to a new parent.
   If childId and newParentId are not provided, it will prompt for them interactively.
   Examples:
     sattchel tracker goals move <childId> <newParentId>
     sattchel tracker goals move
     `,
		Args:         cobra.MaximumNArgs(2),
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			pid := getActiveProjectID(cmd, cfg, projectID)
			if pid == "" {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			goals, err := service.GetGoals(cmd.Context(), pid)
			if err != nil || len(goals) == 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			// Identify rootGoalID
			var rootGoalID string
			for _, g := range goals {
				if g.Parent == nil || g.Parent.TargetID == "" {
					rootGoalID = g.ID
					break
				}
			}

			if len(args) == 0 {
				var completions []string
				for _, g := range goals {
					if g.ID == rootGoalID {
						continue
					}
					completions = append(completions, fmt.Sprintf("%s\t%s", g.ID, g.Name))
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			if len(args) == 1 {
				childID := args[0]
				roots := buildGoalTree(goals)
				excludedIDs := getExcludedSubtreeIDs(roots, childID)

				var completions []string
				for _, g := range goals {
					if excludedIDs[g.ID] {
						continue
					}
					completions = append(completions, fmt.Sprintf("%s\t%s", g.ID, g.Name))
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveNoFileComp
		},
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

			var childID string
			var newParentID string

			if len(args) >= 1 {
				childID = args[0]
			}
			if len(args) == 2 {
				newParentID = args[1]
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

			var rootGoalID string
			for _, g := range goals {
				if g.Parent == nil || g.Parent.TargetID == "" {
					rootGoalID = g.ID
					break
				}
			}

			if childID != "" && childID == rootGoalID {
				return fmt.Errorf("the root goal cannot be moved")
			}

			if childID == "" {
				roots := buildGoalTree(goals)
				options := buildSelectOptions(roots, nil, rootGoalID)

				err = huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select Goal to Move").
							Options(options...).
							Value(&childID).
							Validate(func(val string) error {
								if val == rootGoalID {
									return fmt.Errorf("the root goal cannot be moved")
								}
								return nil
							}),
					),
				).WithShowHelp(true).Run()
				if err != nil {
					return err
				}
			}

			if newParentID == "" {
				roots := buildGoalTree(goals)
				excludedIDs := getExcludedSubtreeIDs(roots, childID)

				options := buildSelectOptions(roots, func(g *core.Goal) (bool, bool) {
					if excludedIDs[g.ID] {
						return false, false
					}
					return true, true
				}, "")

				if len(options) == 0 {
					return fmt.Errorf("no valid destination goals available to set as parent")
				}

				err = huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select New Parent Goal").
							Options(options...).
							Value(&newParentID),
					),
				).WithShowHelp(true).Run()
				if err != nil {
					return err
				}
			}

			var movedGoal *core.Goal
			if err := spinner.
				New().
				Title("Moving goal ...").
				Action(func() {
					movedGoal, err = service.ChangeParent(cmd.Context(), pid, childID, newParentID, core.GoalOptions{})
				}).Run(); err != nil {
				return err
			}
			if err != nil {
				return err
			}

			fmt.Printf("Goal %q (%s) moved successfully under parent %s\n", movedGoal.Name, movedGoal.ID, newParentID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func visualizeGoals(service *core.Service, cfg *Config) *cobra.Command {
	projectID := ""

	cmd := &cobra.Command{
		Use:   "visualize",
		Short: "Start the visualizer web server for goals",
		Long: `Start an ephemeral local web server to visualize goals as an interactive mind map.
   Automatically opens the mind map in your default browser.
   Examples:
     sattchel tracker goals visualize
     `,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
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

			fmt.Println("Getting goals ...")
			goals, err := service.GetGoals(cmd.Context(), pid)
			if err != nil {
				return err
			}
			if len(goals) == 0 {
				return fmt.Errorf("no goals found for project %s", pid)
			}

			fmt.Println("Starting visualizer server ...")
			url, shutdown, err := visualizer.StartServer(cmd.Context(), goals, service, pid)
			if err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			fmt.Printf("Visualizer server running at: %s\n", url)
			fmt.Println("Opening in browser...")
			_ = openBrowser(url)

			fmt.Println("Press Ctrl+C to stop the visualizer server.")

			// Wait for interrupt signal to stop the server
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt)
			<-stop

			fmt.Println("\nStopping server ...")
			if err := shutdown(); err != nil {
				fmt.Printf("Error stopping server: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goals. If not provided, the default project will be used")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // "linux", "freebsd", etc.
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Run()
}
