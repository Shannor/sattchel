package driving

import (
	"fmt"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"slices"
	"strings"

	"sattchel/pkg/loader"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/spf13/cobra"
)

func goals(service *core.Service, cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "goals [verb]",
		Short:   "Manage goals",
		Aliases: []string{"g"},
		Long: `Commands to manage goals.
     Examples:
       satt tracker goals add <name>
       satt tracker goals set
       satt tracker goals list
       satt tracker goals move <childId> <newParentId>
       satt tracker goals update <id>
       satt tracker goals view <id>
       `,
	}
	cmd.AddCommand(addGoal(service, cfg))
	cmd.AddCommand(setGoal(service, cfg))
	cmd.AddCommand(listGoals(service, cfg))
	cmd.AddCommand(moveGoal(service, cfg))
	cmd.AddCommand(viewGoal(service, cfg))
	cmd.AddCommand(updateGoal(service, cfg))
	return cmd
}

func addGoal(service *core.Service, cfg *Config) *cobra.Command {
	description := ""
	parentID := ""
	projectID := ""
	impact := ""
	effort := ""
	relationship := ""
	memberID := ""
	changeCurrent := false
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new goal",
		Long: `Add a new goal.
	Will create a new goal. If it's the root goal it will automatically get set as current'.
	For each goal after it will stay pointing at root unless you provide a parent or flag on creation to change it.
   Examples:
     satt tracker goal add short
     satt tracker goal add "Long Title with Spaces"
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
				Effort:      core.Effort(effort),
				Impact:      core.Impact(impact),
				MemberID:    memberID,
			}
			goal, err := service.CreateGoal(cmd.Context(), pid, args[0], options)
			if err != nil {
				return err
			}
			fmt.Printf("Goal %s created successfully\n", goal.Name)
			if changeCurrent || !goal.HasParent() {
				_ = cfg.SetCurrentGoalID(goal.ID)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the goal")
	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	cmd.Flags().StringVarP(&parentID, "parent", "", "", "Parent goal id of the goal. If not provided, the last parent will be used")
	cmd.Flags().BoolVarP(&changeCurrent, "set", "s", false, "Set the newly created goal as current")
	cmd.Flags().StringVarP(&effort, "effort", "e", string(core.UnknownEffort), "How much effort is required to achieve the goal")
	cmd.Flags().StringVarP(&impact, "impact", "i", string(core.UnknownImpact), "How much impact will the goal have")
	cmd.Flags().StringVarP(&relationship, "relationship", "r", string(core.LinkPreferred), "Requirement relationship with parent goal")
	cmd.Flags().StringVarP(&memberID, "memberId", "m", "", "(Optional) Member ID to assign to the goal")

	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("parent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		pid := getActiveProjectID(cmd, cfg, projectID)
		return getGoalCompletions(service, pid), cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("effort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			string(core.LowEffort),
			string(core.MediumEffort),
			string(core.HighEffort),
		}, cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("impact", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			string(core.LowImpact),
			string(core.MediumImpact),
			string(core.HighImpact),
		}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("relationship", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			string(core.LinkOptional),
			string(core.LinkPreferred),
			string(core.LinkRequired),
		}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("memberId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getMemberCompletions(service), cobra.ShellCompDirectiveNoFileComp
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

			err = loader.Run("Getting goals ...", func() {
				goals, err = service.GetGoals(cmd.Context(), pid)
			})
			if err != nil {
				return err
			}

			if len(goals) == 0 {
				return fmt.Errorf("no goals found for project %s", pid)
			}

			currentGoalID := cfg.CurrentGoalID()
			selectedID, err := tui.ChooseGoal(goals, "Select Active Goal", currentGoalID, nil, nil)
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

			err = loader.Run("Getting goals ...", func() {
				goals, err = service.GetGoals(cmd.Context(), pid)
			})
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
     satt tracker goals move <childId> <newParentId>
     satt tracker goals move
     `,
		Args:         cobra.MaximumNArgs(2),
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			pid := getActiveProjectID(cmd, cfg, projectID)
			if pid == "" {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			// TODO: This logic of where a child can move too should be decided in the service layer.
			// The UI shouldn't make this choice.
			goals, err := service.GetGoals(cmd.Context(), pid)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 0 {
				var completions []string
				for _, g := range goals {
					if g.IsRoot() {
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
			err = loader.Run("Getting goals ...", func() {
				goals, err = service.GetGoals(cmd.Context(), pid)
			})
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
				childID, err = tui.ChooseGoal(goals, "Select Goal to Move", "", nil, func(val string) error {
					if val == rootGoalID {
						return fmt.Errorf("the root goal cannot be moved")
					}
					return nil
				})
				if err != nil {
					return err
				}
			}

			if newParentID == "" {
				roots := buildGoalTree(goals)
				excludedIDs := getExcludedSubtreeIDs(roots, childID)

				newParentID, err = tui.ChooseGoal(goals, "Select New Parent Goal", "", func(g *core.Goal) (bool, bool) {
					if excludedIDs[g.ID] {
						return false, false
					}
					return true, true
				}, nil)
				if err != nil {
					return err
				}
			}

			var movedGoal *core.Goal
			err = loader.Run("Moving goal ...", func() {
				movedGoal, err = service.ChangeParent(cmd.Context(), pid, childID, newParentID, core.GoalOptions{})
			})
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

func viewGoal(service *core.Service, cfg *Config) *cobra.Command {
	projectID := ""

	cmd := &cobra.Command{
		Use:   "view [id]",
		Short: "View goal details",
		Long: `View detailed information about a tracker goal.
   If no ID is provided, an interactive select interface will be displayed.
   Examples:
     satt tracker goals view <id>
     satt tracker goals view
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			pid := getActiveProjectID(cmd, cfg, projectID)
			return getGoalCompletions(service, pid), cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := getActiveProjectID(cmd, cfg, projectID)
			if pid == "" {
				return fmt.Errorf("no project selected")
			}

			var selectedID string
			var err error
			if len(args) > 0 {
				selectedID = args[0]
			} else {
				var goals []core.Goal
				err = loader.Run("Getting goals ...", func() {
					goals, err = service.GetGoals(cmd.Context(), pid)
				})
				if err != nil {
					return err
				}
				if len(goals) == 0 {
					return fmt.Errorf("no goals found for project %s", pid)
				}

				currentGoalID := cfg.CurrentGoalID()
				selectedID, err = tui.ChooseGoal(goals, "Select Goal to View", currentGoalID, nil, nil)
				if err != nil {
					return err
				}
			}

			if selectedID == "" {
				return fmt.Errorf("no goal selected")
			}

			var targetGoal *core.Goal
			err = loader.Run("Getting goal details ...", func() {
				targetGoal, err = service.GetGoal(cmd.Context(), selectedID)
			})
			if err != nil {
				return err
			}

			fmt.Print(tui.RenderGoalDetails(targetGoal))
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func updateGoal(service *core.Service, cfg *Config) *cobra.Command {
	var (
		name        string
		description string
		effort      string
		impact      string
		memberID    string
		status      string
	)

	cmd := &cobra.Command{
		Use:          "update <id>",
		Aliases:      []string{"edit"},
		Short:        "Update a goal's details",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			// Check if at least one flag was set
			if !cmd.Flags().Changed("name") &&
				!cmd.Flags().Changed("description") &&
				!cmd.Flags().Changed("effort") &&
				!cmd.Flags().Changed("impact") &&
				!cmd.Flags().Changed("memberId") &&
				!cmd.Flags().Changed("status") {
				return fmt.Errorf("at least one flag must be specified for update")
			}

			options := core.GoalOptions{
				Description: description,
				Effort:      core.Effort(effort),
				Impact:      core.Impact(impact),
				MemberID:    memberID,
				Status:      core.GoalStatus(status),
			}

			var goal *core.Goal
			var err error
			runErr := loader.Run("Updating goal...", func() {
				goal, err = service.UpdateGoal(cmd.Context(), id, name, options)
			})
			if runErr != nil {
				return runErr
			}
			if err != nil {
				return err
			}

			fmt.Printf("Goal %q (%s) updated successfully\n", goal.Name, goal.ID)
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			pid := getActiveProjectID(cmd, cfg, "")
			return getGoalCompletions(service, pid), cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New name of the goal")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description of the goal")
	cmd.Flags().StringVarP(&effort, "effort", "e", "", "New effort level of the goal")
	cmd.Flags().StringVarP(&impact, "impact", "i", "", "New impact level of the goal")
	cmd.Flags().StringVarP(&memberID, "memberId", "m", "", "New member ID assigned to the goal")
	cmd.Flags().StringVar(&status, "status", "", "New status of the goal")

	_ = cmd.RegisterFlagCompletionFunc("effort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			string(core.LowEffort),
			string(core.MediumEffort),
			string(core.HighEffort),
		}, cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("impact", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			string(core.LowImpact),
			string(core.MediumImpact),
			string(core.HighImpact),
		}, cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("memberId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getMemberCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			string(core.GoalOpen),
			string(core.GoalInProgress),
			string(core.GoalCompleted),
			string(core.GoalCancelled),
			string(core.GoalDraft),
		}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
