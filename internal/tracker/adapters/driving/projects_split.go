package driving

import (
	"fmt"
	"strings"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func splitProjectCmd(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "split [source_project_id]",
		Short: "Split or move a goal and its descendants to a new or existing project",
		Long: `Split or move a goal. You choose a source project, a goal to split/move, and either a new project name or an existing target project.
The split goal and all of its descendant goals, along with member assignments, will be moved to the target project.
If splitting to a new project, the goal will become the root goal of that new project.
If moving to an existing project, by default the goal is attached under its root goal (or a specified parent goal).
   Examples:
     satt tracker project split <source_project_id> --goal <goal_id> --new <new_project_name>
     satt tracker project split <source_project_id> --goal <goal_id> --to <target_project_id>
     satt tracker project split <source_project_id> --goal <goal_id> --to <target_project_id> --parent <target_parent_goal_id>
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			splitGoalID, _ := cmd.Flags().GetString("goal")
			targetProjectID, _ := cmd.Flags().GetString("to")
			targetParentGoalID, _ := cmd.Flags().GetString("parent")
			newProjectName, _ := cmd.Flags().GetString("new")

			if targetProjectID != "" && newProjectName != "" {
				return fmt.Errorf("cannot specify both --new and --to")
			}

			sourceProjID := ""
			if len(args) > 0 {
				sourceProjID = args[0]
			}
			if sourceProjID == "" {
				var err error
				sourceProjID, err = ensureProjectID(cmd, service, cfg, "")
				if err != nil {
					return err
				}
			}

			// If goal is missing, prompt or fail
			if splitGoalID == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("goal ID (--goal) is required in non-interactive mode")
				}
				goals, err := service.GetGoals(cmd.Context(), sourceProjID)
				if err != nil {
					return err
				}
				if len(goals) == 0 {
					return fmt.Errorf("selected project has no goals to split/move")
				}

				splitGoalID, err = tui.ChooseGoal(goals, "Select Goal to Split/Move (this goal and its children will move)", "", nil, nil)
				if err != nil {
					return err
				}
			}

			// If target destination is missing, prompt or fail
			if targetProjectID == "" && newProjectName == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("target project (--to) or new project name (--new) is required in non-interactive mode")
				}
				actionOpts := []tui.ListOption{
					{TitleStr: "Split into a new project", DescriptionStr: "Create a new project for this goal hierarchy", ValueStr: "new"},
					{TitleStr: "Move to an existing project", DescriptionStr: "Move goal hierarchy under an existing project", ValueStr: "existing"},
				}
				selectedAction, err := tui.Choose("Select Destination Type", actionOpts)
				if err != nil {
					return err
				}
				if selectedAction == nil {
					return fmt.Errorf("no destination type selected")
				}
				action := selectedAction.ValueStr

				if action == "new" {
					err = tui.NewForm(
						huh.NewGroup(
							huh.NewInput().
								Title("New Project Name").
								Value(&newProjectName).
								Validate(func(str string) error {
									if strings.TrimSpace(str) == "" {
										return fmt.Errorf("project name is required")
									}
									return nil
								}),
						),
					).Run()
					if err != nil {
						return err
					}
				} else {
					projects, err := service.GetProjects(cmd.Context())
					if err != nil {
						return err
					}
					if len(projects) <= 1 {
						return fmt.Errorf("need at least one other project to move goal to")
					}

					var targetProjects []core.Project
					for _, p := range projects {
						if p.ID == sourceProjID {
							continue
						}
						targetProjects = append(targetProjects, p)
					}

					targetProjectID, err = tui.ChooseProject(targetProjects, "Select Target Project", "")
					if err != nil {
						return err
					}
				}
			}

			if newProjectName == "" && sourceProjID == targetProjectID {
				return fmt.Errorf("source and target projects must be different")
			}

			// Interactive target parent selection if not provided and destination is an existing project
			if newProjectName == "" && targetParentGoalID == "" && !cmd.Flags().Changed("parent") && loader.IsTerminal() {
				targetGoals, err := service.GetGoals(cmd.Context(), targetProjectID)
				if err == nil && len(targetGoals) > 0 {
					attachOpts := []tui.ListOption{
						{TitleStr: "No, attach under the root goal", DescriptionStr: "Root goal of target project", ValueStr: "false"},
						{TitleStr: "Yes, select a parent goal", DescriptionStr: "Choose specific parent in target project", ValueStr: "true"},
					}
					selectedAttach, err := tui.Choose("Attach goal under a specific parent in target project?", attachOpts)
					if err != nil {
						return err
					}
					if selectedAttach == nil {
						return fmt.Errorf("no attach option selected")
					}
					if selectedAttach.ValueStr == "true" {
						targetParentGoalID, err = tui.ChooseGoal(targetGoals, "Select Target Parent Goal", "", nil, nil)
						if err != nil {
							return err
						}
					}
				}
			}

			var (
				tgtProj *core.Project
				err     error
			)
			_ = loader.Run("Splitting/moving goal...", func() {
				tgtProj, err = service.SplitProject(cmd.Context(), sourceProjID, targetProjectID, splitGoalID, targetParentGoalID, newProjectName)
			})
			if err != nil {
				return err
			}

			if newProjectName != "" {
				writer.Success(fmt.Sprintf("Project split successfully. New project %s created.", tgtProj.Label))
			} else {
				writer.Success(fmt.Sprintf("Goal moved successfully to project %s.", tgtProj.Label))
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().String("goal", "", "Goal ID that needs to be split/moved")
	cmd.Flags().String("to", "", "Target project ID (for existing project)")
	cmd.Flags().String("parent", "", "Parent goal ID in target project (for existing project)")
	cmd.Flags().String("new", "", "Name of the new project (for splitting/creating new)")

	_ = cmd.RegisterFlagCompletionFunc("goal", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		sourceProjID := ""
		if len(args) > 0 {
			sourceProjID = args[0]
		} else {
			sourceProjID = cfg.CurrentProjectID()
		}
		return getGoalCompletions(service, sourceProjID), cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("to", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("parent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		toProjID, _ := cmd.Flags().GetString("to")
		return getGoalCompletions(service, toProjID), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
