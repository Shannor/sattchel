package driving

import (
	"fmt"
	"slices"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

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
