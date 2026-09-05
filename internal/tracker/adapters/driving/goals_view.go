package driving

import (
	"fmt"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

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
