package driving

import (
	"context"
	"errors"
	"fmt"
	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"
	"sattchel/pkg/set"
	"slices"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/spf13/cobra"
)

var chooseGoalForDelete = promptChooseGoalForDelete
var confirmRecursiveGoalDelete = promptConfirmRecursiveGoalDelete

func promptChooseGoalForDelete(goals []core.Goal, title string, currentGoalID string, filterFn func(*core.Goal) bool, validateFn func(string) error) (string, error) {
	if !loader.IsTerminal() {
		return "", fmt.Errorf("goal ID is required in non-interactive mode")
	}
	return tui.ChooseGoal(goals, title, currentGoalID, filterFn, validateFn)
}

func promptConfirmRecursiveGoalDelete(goal *core.Goal) (bool, error) {
	if !loader.IsTerminal() {
		return true, nil
	}

	title := fmt.Sprintf("Delete goal %q?", goal.Name)
	if goal.HasChildren() {
		title = fmt.Sprintf("Delete goal %q and its descendants?", goal.Name)
	}

	confirmed := false
	err := tui.NewForm(
		huh.NewGroup(
			huh.NewSelect[bool]().
				Title(title).
				Options(
					huh.NewOption("Yes", true),
					huh.NewOption("No", false),
				).
				Value(&confirmed),
		),
	).Run()
	return confirmed, err
}

func goals(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
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
       satt tracker goals delete <id>
       satt tracker goals view <id>
       satt tracker goals triage
       `,
	}
	cmd.AddCommand(addGoal(service, cfg, writer))
	cmd.AddCommand(setGoal(service, cfg, writer))
	cmd.AddCommand(listGoals(service, cfg, writer))
	cmd.AddCommand(moveGoal(service, cfg, writer))
	cmd.AddCommand(deleteGoal(service, cfg, writer))
	cmd.AddCommand(viewGoal(service, cfg, writer))
	cmd.AddCommand(updateGoal(service, cfg, writer))
	cmd.AddCommand(triageGoals(service, cfg, writer))
	return cmd
}

func addGoal(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	description := ""
	parentID := ""
	projectID := ""
	impact := ""
	effort := ""
	relationship := ""
	memberID := ""
	changeCurrent := false
	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new goal",
		Long: `Add a new goal.
	Will create a new goal. If it's the root goal it will automatically get set as current'.
	For each goal after it will stay pointing at root unless you provide a parent or flag on creation to change it.

	If no name is provided, an interactive form will be shown.

   Examples:
     satt tracker goal add short
     satt tracker goal add "Long Title with Spaces"
     satt tracker goal add
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := ensureProjectID(cmd, service, cfg, projectID)
			if err != nil {
				return err
			}

			// If no name provided, show interactive form
			if len(args) == 0 {
				if !loader.IsTerminal() {
					return fmt.Errorf("goal name is required in non-interactive mode")
				}
				return addGoalInteractive(cmd.Context(), service, cfg, pid)
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
			writer.Success(fmt.Sprintf("Goal %s (%s) created successfully", goal.Name, goal.ID))
			if changeCurrent || !goal.HasParent() {
				_ = cfg.SetCurrentGoalID(goal.ID)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. Default: default project")
	cmd.Flags().StringVarP(&parentID, "parent", "", "", "Parent goal id of the goal. Default: last parent")
	cmd.Flags().StringVarP(&description, "description", "d", "", "(Optional) Description of the goal")
	cmd.Flags().BoolVarP(&changeCurrent, "set", "s", false, "(Optional) Set the newly created goal as current")
	cmd.Flags().StringVarP(&effort, "effort", "e", string(core.UnknownEffort), "(Optional) How much effort is required to achieve the goal")
	cmd.Flags().StringVarP(&impact, "impact", "i", string(core.UnknownImpact), "(Optional) How much impact will the goal have")
	cmd.Flags().StringVarP(&relationship, "relationship", "r", string(core.LinkOptional), "relationship with parent goal. Default: optional")
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
			string(core.LinkRequired),
		}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("memberId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getMemberCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func addGoalInteractive(ctx context.Context, service *core.Service, cfg *Config, pid string) error {
	// Fetch available goals for parent selection
	var goals []core.Goal
	var err error
	_ = loader.Run("Loading goals ...", func() {
		goals, err = service.GetGoals(ctx, pid)
	})
	if err != nil {
		return err
	}

	// Fetch available members
	var members []core.Member
	_ = loader.Run("Loading members ...", func() {
		members, err = service.GetMembers(ctx)
	})
	if err != nil {
		return err
	}

	// Determine default parent
	defaultParent := ""
	if lastGoal := cfg.CurrentGoalID(); lastGoal != "" {
		defaultParent = lastGoal
	}

	var (
		goalName        string
		description     string
		parentID        string
		impactVal       string
		effortVal       string
		memberIDVal     string
		relationshipVal string
		setCurrent      bool
	)

	form := tui.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Goal Name").Prompt(":").Validate(func(val string) error {
				if val == "" {
					return fmt.Errorf("goal name is required")
				}
				return nil
			}).Value(&goalName),
		).WithHeight(12),
		huh.NewGroup(
			huh.NewInput().Title("Description").Prompt(":").Placeholder("(Optional)").Value(&description),
		).WithHeight(12),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if len(goals) > 0 {
		selectedParent, err := tui.ChooseGoal(goals, "Select Parent Goal", defaultParent, nil, nil)
		if err != nil {
			if !errors.Is(err, tui.ErrUserAborted) {
				return err
			}
		} else {
			parentID = selectedParent
		}
	}

	impactOpts := []tui.ListOption{
		{TitleStr: "Unknown", DescriptionStr: "Not assessed yet", ValueStr: string(core.UnknownImpact)},
		{TitleStr: "Low", DescriptionStr: "Low impact on overall project", ValueStr: string(core.LowImpact)},
		{TitleStr: "Medium", DescriptionStr: "Medium impact on project", ValueStr: string(core.MediumImpact)},
		{TitleStr: "High", DescriptionStr: "High impact on project", ValueStr: string(core.HighImpact)},
	}
	selectedImpact, err := tui.Choose("Select Impact", impactOpts)
	if err != nil {
		if !errors.Is(err, tui.ErrUserAborted) {
			return err
		}
	} else {
		impactVal = selectedImpact.ValueStr
	}

	effortOpts := []tui.ListOption{
		{TitleStr: "Unknown", DescriptionStr: "Not assessed yet", ValueStr: string(core.UnknownEffort)},
		{TitleStr: "Low", DescriptionStr: "Quick task / low effort", ValueStr: string(core.LowEffort)},
		{TitleStr: "Medium", DescriptionStr: "Moderate effort required", ValueStr: string(core.MediumEffort)},
		{TitleStr: "High", DescriptionStr: "Significant effort required", ValueStr: string(core.HighEffort)},
	}
	selectedEffort, err := tui.Choose("Select Effort", effortOpts)
	if err != nil {
		if !errors.Is(err, tui.ErrUserAborted) {
			return err
		}
	} else {
		effortVal = selectedEffort.ValueStr
	}

	if len(members) > 0 {
		selectedMember, err := tui.ChooseMember(members, "Select Assigned Member", true)
		if err != nil {
			if !errors.Is(err, tui.ErrUserAborted) {
				return err
			}
		} else {
			memberIDVal = selectedMember
		}
	}

	if parentID != "" {
		relOpts := []tui.ListOption{
			{TitleStr: "Optional", DescriptionStr: "Child goal is optional", ValueStr: string(core.LinkOptional)},
			{TitleStr: "Required", DescriptionStr: "Child goal is required before parent completion", ValueStr: string(core.LinkRequired)},
		}
		selectedRel, err := tui.Choose("Select Link Relationship", relOpts)
		if err != nil {
			if !errors.Is(err, tui.ErrUserAborted) {
				return err
			}
		} else {
			relationshipVal = selectedRel.ValueStr
		}
	}

	if err := tui.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Set as current goal?").Value(&setCurrent),
		).WithHeight(6),
	).Run(); err != nil {
		if !errors.Is(err, huh.ErrUserAborted) {
			return err
		}
	}

	options := core.GoalOptions{
		ParentID:         parentID,
		Description:      description,
		Effort:           core.Effort(effortVal),
		Impact:           core.Impact(impactVal),
		MemberID:         memberIDVal,
		LinkRelationship: core.LinkRelationship(relationshipVal),
	}

	goal, err := service.CreateGoal(ctx, pid, goalName, options)
	if err != nil {
		return err
	}

	if setCurrent || !goal.HasParent() {
		_ = cfg.SetCurrentGoalID(goal.ID)
	}

	fmt.Printf("Goal %s (%s) created successfully\n", goal.Name, goal.ID)
	return nil
}

func setGoal(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	projectID := ""

	cmd := &cobra.Command{
		Use:          "set [id]",
		Short:        "Set Active Goal",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := ensureProjectID(cmd, service, cfg, projectID)
			if err != nil {
				return err
			}

			var (
				goals []core.Goal
			)

			_ = loader.Run("Getting goals ...", func() {
				goals, err = service.GetGoals(cmd.Context(), pid)
			})
			if err != nil {
				return err
			}

			if len(goals) == 0 {
				return fmt.Errorf("no goals found for project %s", pid)
			}

			selectedID := ""
			if len(args) > 0 {
				selectedID = args[0]
			}

			if selectedID == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("goal ID is required in non-interactive mode")
				}
				currentGoalID := cfg.CurrentGoalID()
				selectedID, err = tui.ChooseGoal(goals, "Select Active Goal", currentGoalID, nil, nil)
				if err != nil {
					return err
				}
			}

			if err := cfg.SetCurrentGoalID(selectedID); err != nil {
				return fmt.Errorf("failed to save active goal: %w", err)
			}

			idx := slices.IndexFunc(goals, func(g core.Goal) bool { return g.ID == selectedID })
			if idx == -1 {
				return fmt.Errorf("goal with ID %q not found in project %s", selectedID, pid)
			}
			g := goals[idx]

			writer.Success(fmt.Sprintf("Active goal set to: %s (%s)", g.Name, g.ID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

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

func moveGoal(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		relationship = core.LinkOptional
		projectID    string
	)

	cmd := &cobra.Command{
		Use:     "move [childId] [newParentId]",
		Short:   "Move a goal to a new parent",
		Aliases: []string{"mv"},
		Long: `Move a goal to a new parent.
   If childId and newParentId are not provided, an interactive prompt will be displayed.
   Examples:
     satt tracker goals move
     satt tracker goals move <childId> <newParentId> -r <relationship>
     `,
		Args:         cobra.MaximumNArgs(2),
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			pid := getActiveProjectID(cmd, cfg, projectID)
			if pid == "" {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

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
					completions = append(completions, cobra.CompletionWithDesc(g.ID, g.Name))
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			if len(args) == 1 {
				childID := args[0]
				allowedParents, err := service.GetAllowedParents(cmd.Context(), pid, childID)
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}

				allowedSet := set.NewFromFunc(allowedParents, func(g core.Goal) string { return g.ID })
				var completions []string
				for _, g := range goals {
					if !allowedSet.Contains(g.ID) {
						continue
					}
					completions = append(completions, cobra.CompletionWithDesc(g.ID, g.Name))
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := ensureProjectID(cmd, service, cfg, projectID)
			if err != nil {
				return err
			}

			var childID string
			var newParentID string

			if len(args) >= 1 {
				childID = args[0]
			}
			if len(args) == 2 {
				newParentID = args[1]
			}

			if (childID == "" || newParentID == "") && !loader.IsTerminal() {
				return fmt.Errorf("both childId and newParentId are required in non-interactive mode")
			}

			var (
				goals []core.Goal
			)
			_ = loader.Run("Getting goals ...", func() {
				goals, err = service.GetGoals(cmd.Context(), pid)
			})
			if err != nil {
				return err
			}
			if len(goals) == 0 {
				return fmt.Errorf("no goals found for project %s", pid)
			}

			rootGoal, err := service.GetRootGoal(cmd.Context(), pid)
			if err != nil {
				return err
			}

			if childID != "" && childID == rootGoal.ID {
				return fmt.Errorf("the root goal cannot be moved")
			}

			if childID == "" {
				currentGoalID := cfg.CurrentGoalID()
				childID, err = tui.ChooseGoal(goals, "Select Goal to Move", currentGoalID, nil, func(val string) error {
					if val == rootGoal.ID {
						return fmt.Errorf("the root goal cannot be moved")
					}
					return nil
				})
				if err != nil {
					return err
				}
			}

			allowedGoals, err := service.GetAllowedParents(cmd.Context(), pid, childID)
			if err != nil {
				return err
			}
			if len(allowedGoals) == 0 {
				return fmt.Errorf("no valid parent goals available to move this goal under")
			}
			allowedSet := set.NewFromFunc(allowedGoals, func(g core.Goal) string { return g.ID })
			if newParentID == "" {
				newParentID, err = tui.ChooseGoal(goals, "Select New Parent Goal", "", func(g *core.Goal) bool {
					return allowedSet.Contains(g.ID)
				}, nil)
				if err != nil {
					return err
				}
			}

			if !cmd.Flags().Changed("relationship") && len(args) == 0 && loader.IsTerminal() {
				relOptions := []tui.ListOption{
					{TitleStr: "Optional", DescriptionStr: "Child goal is optional", ValueStr: string(core.LinkOptional)},
					{TitleStr: "Required", DescriptionStr: "Child goal is required before parent completion", ValueStr: string(core.LinkRequired)},
				}
				selectedRel, err := tui.Choose("Select Link Relationship", relOptions)
				if err != nil {
					return err
				}
				if selectedRel == nil {
					return fmt.Errorf("no relationship selected")
				}
				relationship = core.LinkRelationship(selectedRel.ValueStr)
			}

			var movedGoal *core.Goal
			_ = loader.Run("Moving goal ...", func() {
				movedGoal, err = service.ChangeParent(cmd.Context(), pid, childID, newParentID, core.GoalOptions{
					LinkRelationship: relationship,
				})
			})
			if err != nil {
				return err
			}

			writer.Success(fmt.Sprintf("Goal %q (%s) moved successfully under parent %s", movedGoal.Name, movedGoal.ID, newParentID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	cmd.Flags().StringVarP((*string)(&relationship), "relationship", "r", string(core.LinkOptional), "Relationship of the link between the goal and its new parent")
	_ = cmd.RegisterFlagCompletionFunc("relationship", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{string(core.LinkOptional), string(core.LinkRequired)}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func deleteGoal(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		projectID string
		recursive bool
	)

	cmd := &cobra.Command{
		Use:          "delete [id]",
		Aliases:      []string{"remove", "rm"},
		Short:        "Delete a goal",
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
			pid, err := ensureProjectID(cmd, service, cfg, projectID)
			if err != nil {
				return err
			}

			goalID := ""
			if len(args) > 0 {
				goalID = args[0]
			}

			var (
				goals []core.Goal
			)
			if goalID == "" || recursive {
				_ = loader.Run("Getting goals ...", func() {
					goals, err = service.GetGoals(cmd.Context(), pid)
				})
				if err != nil {
					return err
				}
				if len(goals) == 0 {
					return fmt.Errorf("no goals found for project %s", pid)
				}
			}

			if goalID == "" {
				goalID, err = chooseGoalForDelete(goals, "Select Goal to Delete", cfg.CurrentGoalID(), func(g *core.Goal) bool {
					return !g.IsRoot()
				}, nil)
				if err != nil {
					return err
				}
			}

			var (
				selectedGoal            *core.Goal
				currentGoalNeedsRecheck bool
			)
			if recursive {
				for i := range goals {
					if goals[i].ID == goalID {
						selectedGoal = &goals[i]
						break
					}
				}
				if selectedGoal == nil {
					return fmt.Errorf("goal %s not found in project %s", goalID, pid)
				}

				currentGoalID := cfg.CurrentGoalID()
				if currentGoalID != "" && currentGoalID != goalID {
					currentGoal, getErr := service.GetGoal(cmd.Context(), currentGoalID)
					if getErr == nil && currentGoal != nil && currentGoal.ProjectID == pid {
						currentGoalNeedsRecheck = true
					}
				}

				confirmed, err := confirmRecursiveGoalDelete(selectedGoal)
				if err != nil {
					return err
				}
				if !confirmed {
					writer.Info("Recursive delete cancelled")
					return nil
				}
			}

			runErr := loader.Run("Deleting goal...", func() {
				if recursive {
					err = service.DeleteGoalRecursive(cmd.Context(), pid, goalID)
					return
				}
				err = service.DeleteGoal(cmd.Context(), pid, goalID)
			})
			if runErr != nil {
				return runErr
			}
			if err != nil {
				return err
			}

			if currentGoalID := cfg.CurrentGoalID(); currentGoalID != "" {
				switch {
				case currentGoalID == goalID:
					_ = cfg.SetCurrentGoalID("")
				case recursive && currentGoalNeedsRecheck:
					currentGoal, getErr := service.GetGoal(cmd.Context(), currentGoalID)
					if getErr != nil || currentGoal == nil {
						_ = cfg.SetCurrentGoalID("")
					}
				}
			}

			if recursive {
				writer.Success(fmt.Sprintf("Goal %s and its descendants deleted successfully", goalID))
				return nil
			}
			writer.Success(fmt.Sprintf("Goal %s deleted successfully", goalID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Delete the selected goal and all of its descendants")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func viewGoal(service *core.Service, cfg *Config, _ printer.Writer) *cobra.Command {
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
			pid, err := ensureProjectID(cmd, service, cfg, projectID)
			if err != nil {
				return err
			}

			var (
				selectedID string
				goals      []core.Goal
				parent     *core.Goal
			)
			if len(args) > 0 {
				selectedID = args[0]
			} else {
				if !loader.IsTerminal() {
					return fmt.Errorf("goal ID is required in non-interactive mode")
				}
				_ = loader.Run("Getting goals ...", func() {
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
			_ = loader.Run("Getting goal details ...", func() {
				targetGoal, err = service.GetGoal(cmd.Context(), selectedID)
			})
			if err != nil {
				return err
			}

			if targetGoal.Parent != nil {
				parent, err = service.GetGoal(cmd.Context(), targetGoal.Parent.TargetID)
				if err != nil {
					return err
				}
			}

			fmt.Print(tui.RenderGoalDetails(targetGoal, parent))
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func updateGoal(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		name        string
		description string
		effort      string
		impact      string
		memberID    string
		status      string
		projectID   string
	)

	cmd := &cobra.Command{
		Use:          "update [id]",
		Aliases:      []string{"edit"},
		Short:        "Update a goal's details",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) > 0 {
				id = args[0]
			}

			if id == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("goal ID is required in non-interactive mode")
				}
				pid, err := ensureProjectID(cmd, service, cfg, projectID)
				if err != nil {
					return err
				}
				var goals []core.Goal
				_ = loader.Run("Getting goals ...", func() {
					goals, err = service.GetGoals(cmd.Context(), pid)
				})
				if err != nil {
					return err
				}
				if len(goals) == 0 {
					return fmt.Errorf("no goals found for project %s", pid)
				}
				selectedID, err := tui.ChooseGoal(goals, "Select Goal to Update", cfg.CurrentGoalID(), nil, nil)
				if err != nil {
					return err
				}
				id = selectedID
			}

			hasFlags := cmd.Flags().Changed("name") ||
				cmd.Flags().Changed("description") ||
				cmd.Flags().Changed("effort") ||
				cmd.Flags().Changed("impact") ||
				cmd.Flags().Changed("memberId") ||
				cmd.Flags().Changed("status")

			if !hasFlags {
				if !loader.IsTerminal() {
					return fmt.Errorf("at least one flag must be specified for update in non-interactive mode")
				}
				updatedGoal, err := updateGoalInteractive(cmd.Context(), service, id)
				if err != nil {
					return err
				}
				writer.Success(fmt.Sprintf("Goal %q (%s) updated successfully", updatedGoal.Name, updatedGoal.ID))
				return nil
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
			_ = loader.Run("Updating goal...", func() {
				goal, err = service.UpdateGoal(cmd.Context(), id, name, options)
			})
			if err != nil {
				return err
			}

			writer.Success(fmt.Sprintf("Goal %q (%s) updated successfully", goal.Name, goal.ID))
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			pid := getActiveProjectID(cmd, cfg, projectID)
			return getGoalCompletions(service, pid), cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. Default: active project")
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

func updateGoalInteractive(ctx context.Context, service *core.Service, goalID string) (*core.Goal, error) {
	var currentGoal *core.Goal
	var err error
	_ = loader.Run("Fetching goal details...", func() {
		currentGoal, err = service.GetGoal(ctx, goalID)
	})
	if err != nil {
		return nil, err
	}

	name := currentGoal.Name
	description := currentGoal.Description
	effortVal := string(currentGoal.Effort)
	impactVal := string(currentGoal.Impact)
	memberIDVal := ""
	if currentGoal.Member != nil {
		memberIDVal = currentGoal.Member.ID
	}
	statusVal := string(currentGoal.Status)

	form := tui.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Goal Name").Value(&name).Validate(func(val string) error {
				if strings.TrimSpace(val) == "" {
					return fmt.Errorf("goal name is required")
				}
				return nil
			}),
			huh.NewInput().Title("Description").Value(&description),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}

	statusOpts := []tui.ListOption{
		{TitleStr: "Draft", ValueStr: string(core.GoalDraft)},
		{TitleStr: "Open", ValueStr: string(core.GoalOpen)},
		{TitleStr: "In Progress", ValueStr: string(core.GoalInProgress)},
		{TitleStr: "Completed", ValueStr: string(core.GoalCompleted)},
		{TitleStr: "Cancelled", ValueStr: string(core.GoalCancelled)},
	}
	if selStatus, err := tui.Choose("Select Status", statusOpts); err == nil && selStatus != nil {
		statusVal = selStatus.ValueStr
	}

	impactOpts := []tui.ListOption{
		{TitleStr: "Unknown", ValueStr: string(core.UnknownImpact)},
		{TitleStr: "Low", ValueStr: string(core.LowImpact)},
		{TitleStr: "Medium", ValueStr: string(core.MediumImpact)},
		{TitleStr: "High", ValueStr: string(core.HighImpact)},
	}
	if selImpact, err := tui.Choose("Select Impact", impactOpts); err == nil && selImpact != nil {
		impactVal = selImpact.ValueStr
	}

	effortOpts := []tui.ListOption{
		{TitleStr: "Unknown", ValueStr: string(core.UnknownEffort)},
		{TitleStr: "Low", ValueStr: string(core.LowEffort)},
		{TitleStr: "Medium", ValueStr: string(core.MediumEffort)},
		{TitleStr: "High", ValueStr: string(core.HighEffort)},
	}
	if selEffort, err := tui.Choose("Select Effort", effortOpts); err == nil && selEffort != nil {
		effortVal = selEffort.ValueStr
	}

	members, _ := service.GetMembers(ctx)
	if len(members) > 0 {
		if selMember, err := tui.ChooseMember(members, "Select Assigned Member", true); err == nil {
			memberIDVal = selMember
		}
	}

	options := core.GoalOptions{
		Description: description,
		Effort:      core.Effort(effortVal),
		Impact:      core.Impact(impactVal),
		MemberID:    memberIDVal,
		Status:      core.GoalStatus(statusVal),
	}

	var updated *core.Goal
	_ = loader.Run("Updating goal...", func() {
		updated, err = service.UpdateGoal(ctx, goalID, name, options)
	})
	return updated, err
}
