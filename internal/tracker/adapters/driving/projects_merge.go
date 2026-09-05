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

func mergeProjectsCmd(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var parentGoalID string

	cmd := &cobra.Command{
		Use:   "merge [source_project_id] [merge_project_id]",
		Short: "Merge another project into the source project",
		Long: `Merge two projects. The merge project will be absorbed into the source project.
The root goal of the merge project will be attached under the specified parent goal (or the root goal of the source project by default).
All goals and member associations will be moved and preserved. The merge project will be deleted.
   Examples:
     satt tracker project merge <source_project_id> <merge_project_id>
     satt tracker project merge <source_project_id> <merge_project_id> --parent-goal-id <goal_id>
     `,
		Args:         cobra.MaximumNArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var sourceProjID string
			var mergeProjID string

			if len(args) > 0 {
				sourceProjID = args[0]
			}
			if len(args) > 1 {
				mergeProjID = args[1]
			}

			// Interactive mode if arguments are missing
			if sourceProjID == "" || mergeProjID == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("both source_project_id and merge_project_id are required in non-interactive mode")
				}
				projects, err := service.GetProjects(cmd.Context())
				if err != nil {
					return err
				}
				if len(projects) < 2 {
					return fmt.Errorf("need at least 2 projects to perform a merge, got %d", len(projects))
				}

				if sourceProjID == "" {
					sourceProjID, err = tui.ChooseProject(projects, "Select Source Project (the project to merge INTO)", "")
					if err != nil {
						return err
					}
				}

				// Filter out the source project for the merge options
				var remainingProjects []core.Project
				for _, p := range projects {
					if p.ID != sourceProjID {
						remainingProjects = append(remainingProjects, p)
					}
				}

				if mergeProjID == "" {
					mergeProjID, err = tui.ChooseProject(remainingProjects, "Select Project to Merge (the project that will be absorbed)", "")
					if err != nil {
						return err
					}
				}
			}

			if sourceProjID == mergeProjID {
				return fmt.Errorf("cannot merge a project into itself")
			}

			// Interactive parent selection if parentGoalID is empty and flag was not explicitly set
			if parentGoalID == "" && !cmd.Flags().Changed("parent-goal-id") && loader.IsTerminal() {
				sourceGoals, err := service.GetGoals(cmd.Context(), sourceProjID)
				if err == nil && len(sourceGoals) > 0 {
					attachOpts := []tui.ListOption{
						{TitleStr: "No, attach under the root goal", DescriptionStr: "Root goal of source project", ValueStr: "false"},
						{TitleStr: "Yes, select a parent goal", DescriptionStr: "Choose specific parent in source project", ValueStr: "true"},
					}
					selectedAttach, err := tui.Choose("Attach under a specific parent goal in source project?", attachOpts)
					if err != nil {
						return err
					}
					if selectedAttach == nil {
						return fmt.Errorf("no attach option selected")
					}
					if selectedAttach.ValueStr == "true" {
						parentGoalID, err = tui.ChooseGoal(sourceGoals, "Select Parent Goal in Source Project", "", nil, nil)
						if err != nil {
							return err
						}
					}
				}
			}

			// Perform merge
			var runErr error
			_ = loader.Run("Merging projects...", func() {
				runErr = service.MergeProjects(cmd.Context(), sourceProjID, mergeProjID, parentGoalID)
			})
			if runErr != nil {
				return runErr
			}

			if err := cfg.SetCurrentProjectID(sourceProjID); err != nil {
				return fmt.Errorf("failed to save active project: %w", err)
			}

			sourceProj, err := service.GetProject(cmd.Context(), sourceProjID)
			if err != nil {
				writer.Success("Projects merged successfully")
			} else {
				writer.Success(fmt.Sprintf("Projects merged successfully. Active project set to: %s (%s)", sourceProj.Label, sourceProj.ID))
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 {
				// filter completions to not include the first arg
				completions := getProjectCompletions(service)
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

	cmd.Flags().StringVar(&parentGoalID, "parent-goal-id", "", "Parent goal ID where the merge project should land")
	_ = cmd.RegisterFlagCompletionFunc("parent-goal-id", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Complete with goals from the source project (which is the first arg, or current active project)
		sourceProjID := ""
		if len(args) > 0 {
			sourceProjID = args[0]
		} else {
			sourceProjID = cfg.CurrentProjectID()
		}
		return getGoalCompletions(service, sourceProjID), cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
