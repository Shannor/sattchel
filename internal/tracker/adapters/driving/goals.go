package driving

import (
	"fmt"
	"sattchel/internal/tracker/core"
	"slices"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
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

			var (
				selectedID string
				options    []huh.Option[string]
			)

			currentGoalID := cfg.CurrentGoalID()
			for _, goal := range goals {
				option := huh.NewOption(goal.Name, goal.ID)
				if goal.ID == currentGoalID {
					option = option.Selected(true)
					selectedID = goal.ID
				}
				options = append(options, option)
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
			for _, goal := range goals {
				currentMarker := ""
				if goal.ID == currentGoalID {
					currentMarker = " (active)"
				}
				fmt.Printf("- %s: %s%s\n", goal.Name, goal.ID, currentMarker)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	return cmd
}
