package driving

import (
	"fmt"
	"strings"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

func mergeGoals(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "merge [source_goal_id] [merge_goal_id]",
		Short: "Merge another goal into the source goal",
		Long: `Merge two goals. The merge goal will be absorbed and deleted.
All children of the merge goal will be re-parented under the source goal.
   Examples:
     satt tracker goals merge <source_goal_id> <merge_goal_id>
     satt tracker goals merge
     `,
		Args:         cobra.MaximumNArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := ensureProjectID(cmd, service, cfg, projectID)
			if err != nil {
				return err
			}

			var sourceGoalID string
			var mergeGoalID string

			if len(args) > 0 {
				sourceGoalID = args[0]
			}
			if len(args) > 1 {
				mergeGoalID = args[1]
			}

			if (sourceGoalID == "" || mergeGoalID == "") && !loader.IsTerminal() {
				return fmt.Errorf("both source_goal_id and merge_goal_id are required in non-interactive mode")
			}

			var goals []core.Goal
			_ = loader.Run("Getting goals ...", func() {
				goals, err = service.GetGoals(cmd.Context(), pid)
			})
			if err != nil {
				return err
			}
			if len(goals) < 2 {
				return fmt.Errorf("need at least 2 goals in project %s to perform a merge, got %d", pid, len(goals))
			}

			rootGoal, err := service.GetRootGoal(cmd.Context(), pid)
			if err != nil {
				return err
			}

			if sourceGoalID == "" {
				currentGoalID := cfg.CurrentGoalID()
				sourceGoalID, err = tui.ChooseGoal(goals, "Select Source Goal (the goal to merge INTO)", currentGoalID, nil, nil)
				if err != nil {
					return err
				}
			}

			if mergeGoalID == "" {
				mergeGoalID, err = tui.ChooseGoal(goals, "Select Goal to Merge (the goal to absorb & delete)", "", func(g *core.Goal) bool {
					return g.ID != sourceGoalID && g.ID != rootGoal.ID
				}, func(val string) error {
					if val == sourceGoalID {
						return fmt.Errorf("cannot merge a goal into itself")
					}
					if val == rootGoal.ID {
						return fmt.Errorf("the root goal cannot be merged")
					}
					return nil
				})
				if err != nil {
					return err
				}
			}

			if sourceGoalID == mergeGoalID {
				return fmt.Errorf("cannot merge a goal into itself")
			}
			if mergeGoalID == rootGoal.ID {
				return fmt.Errorf("the root goal cannot be merged")
			}

			var sourceGoal *core.Goal
			_ = loader.Run("Merging goals...", func() {
				sourceGoal, err = service.MergeGoals(cmd.Context(), pid, sourceGoalID, mergeGoalID)
			})
			if err != nil {
				return err
			}

			if cfg.CurrentGoalID() == mergeGoalID {
				_ = cfg.SetCurrentGoalID(sourceGoalID)
			}

			writer.Success(fmt.Sprintf("Goal %s deleted and merged successfully into %q (%s)", mergeGoalID, sourceGoal.Name, sourceGoal.ID))
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			pid := getActiveProjectID(cmd, cfg, projectID)
			if len(args) == 0 {
				return getGoalCompletions(service, pid), cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 {
				completions := getGoalCompletions(service, pid)
				var filtered []string
				for _, c := range completions {
					if !strings.HasPrefix(c, args[0]+"\t") {
						filtered = append(filtered, c)
					}
				}
				return filtered, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. Default: active project")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
