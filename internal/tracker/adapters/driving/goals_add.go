package driving

import (
	"context"
	"errors"
	"fmt"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

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
