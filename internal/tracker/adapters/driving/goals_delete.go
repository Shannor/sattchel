package driving

import (
	"fmt"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"charm.land/huh/v2"
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
