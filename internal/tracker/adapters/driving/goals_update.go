package driving

import (
	"context"
	"fmt"
	"strings"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

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
