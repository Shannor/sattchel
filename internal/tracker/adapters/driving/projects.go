package driving

import (
	"fmt"
	"os"
	"strings"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"

	"sattchel/pkg/loader"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func projects(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project [next]",
		Aliases: []string{"p"},
		Short:   "Manage projects",
		Long: `Manage projects.
   Examples:
     satt tracker project create <name> 
     satt tracker project list
     satt tracker project view
     satt tracker project update [id]
     satt tracker project merge <source_project_id> <merge_project_id>
     satt tracker project split <source_project_id>
     `,
	}
	cmd.AddCommand(createProject(service, cfg, writer))
	cmd.AddCommand(listProjects(service, cfg, writer))
	cmd.AddCommand(viewProject(service, cfg, writer))
	cmd.AddCommand(updateProject(service, cfg, writer))
	cmd.AddCommand(mergeProjectsCmd(service, cfg, writer))
	cmd.AddCommand(splitProjectCmd(service, cfg, writer))
	cmd.AddCommand(deleteProjectCmd(service, cfg, writer))
	return cmd
}

func createProject(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	description := ""
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new project",
		Long: `Create a new project.
	If no name is provided, you will be prompted to enter it.
   Examples:
     satt tracker project create <name> 
     satt tracker project create <name> -d "description"
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			if name == "" {
				err := tui.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Project Name").
							Value(&name).
							Validate(func(str string) error {
								if strings.TrimSpace(str) == "" {
									return fmt.Errorf("project name is required")
								}
								return nil
							}),
						huh.NewInput().
							Title("Description").
							Value(&description),
					),
				).Run()
				if err != nil {
					return err
				}
			}

			p, err := service.CreateProject(cmd.Context(), name, description)
			if err != nil {
				return err
			}
			writer.Success(fmt.Sprintf("Project %s created successfully", p.Label))
			_ = cfg.SetCurrentProjectID(p.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "description of the project")
	return cmd
}

func listProjects(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		stdoutFlag  bool
		setActiveID string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Projects and select active project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if setActiveID != "" {
				var (
					projects []core.Project
					err      error
				)

				_ = loader.Run("Getting projects ...", func() {
					projects, err = service.GetProjects(cmd.Context())
				})
				if err != nil {
					return err
				}

				var targetProj *core.Project
				for _, p := range projects {
					if p.ID == setActiveID {
						targetProj = &p
						break
					}
				}
				if targetProj == nil {
					return fmt.Errorf("project with ID %q not found", setActiveID)
				}

				if err := cfg.SetCurrentProjectID(setActiveID); err != nil {
					return fmt.Errorf("failed to save active project: %w", err)
				}
				writer.Success(fmt.Sprintf("Active project set to: %s (%s)", targetProj.Label, targetProj.ID))
				return nil
			}

			var (
				projects []core.Project
				err      error
			)

			err = loader.Run("Getting projects ...", func() {
				projects, err = service.GetProjects(cmd.Context())
			})
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				writer.Info("No projects found")
				return nil
			}

			currentProjID := cfg.CurrentProjectID()
			bypassUI := stdoutFlag || !loader.IsTerminal()
			styles := tui.AutoStyles()

			if bypassUI {
				headers := []string{"Active", "Name", "ID", "Description"}
				var rows [][]string
				for _, project := range projects {
					activeStr := ""
					nameStr := project.Label
					if project.ID == currentProjID {
						activeStr = styles.Success.Render("●")
						nameStr = styles.Success.Bold(true).Render(project.Label)
					} else {
						activeStr = " "
						nameStr = styles.Text.Render(project.Label)
					}

					descVal := project.Description
					if descVal == "" {
						descVal = styles.Muted.Render("-")
					} else {
						descVal = styles.Text.Render(descVal)
					}

					rows = append(rows, []string{
						activeStr,
						nameStr,
						styles.Muted.Render(project.ID),
						descVal,
					})
				}
				fmt.Println(tui.RenderTable(headers, rows))
				return nil
			}

			// Interactive selection
			var options []tui.ListOption
			for _, project := range projects {
				var titleStr string
				var descStr string

				if project.ID == currentProjID {
					titleStr = styles.Success.Bold(true).Render("● " + project.Label + " (active)")
					descStr = styles.Success.Render(project.ID)
				} else {
					titleStr = "  " + project.Label
					descStr = styles.Muted.Render(project.ID)
				}

				if project.Description != "" {
					descStr = descStr + " - " + project.Description
				}

				options = append(options, tui.ListOption{
					TitleStr:       titleStr,
					DescriptionStr: descStr,
					ValueStr:       project.ID,
				})
			}

			selected, err := tui.Choose("Select Active Project", options)
			if err != nil {
				return err
			}

			if selected != nil && selected.ValueStr != "" {
				if err := cfg.SetCurrentProjectID(selected.ValueStr); err != nil {
					return fmt.Errorf("failed to save active project: %w", err)
				}
				var cleanName string
				for _, p := range projects {
					if p.ID == selected.ValueStr {
						cleanName = p.Label
						break
					}
				}
				writer.Success(fmt.Sprintf("Active project set to: %s (%s)", cleanName, selected.ValueStr))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump list directly to stdout instead of interactive UI")
	cmd.Flags().StringVarP(&setActiveID, "set", "s", "", "Set the active project by ID (non-interactive)")
	_ = cmd.RegisterFlagCompletionFunc("set", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func viewProject(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		projectID  string
		stdoutFlag bool
		toFile     string
	)
	cmd := &cobra.Command{
		Use:     "view",
		Aliases: []string{"v"},
		Short:   "View details of a project",
		Long: `View details of a project, including goals, status, and members.
If no projectId is provided, the current active project will be used.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := projectID
			if pid == "" {
				pid = cfg.CurrentProjectID()
			}
			if pid == "" {
				return fmt.Errorf("no active project configured and no --projectId flag provided")
			}

			var (
				project *core.Project
				goals   []core.Goal
				err     error
			)

			err = loader.Run("Retrieving project details...", func() {
				project, err = service.GetProject(cmd.Context(), pid)
				if err != nil {
					return
				}
				goals, err = service.GetGoals(cmd.Context(), pid)
			})
			if err != nil {
				return err
			}

			content := tui.RenderProjectDetails(project, goals)

			if toFile != "" {
				err := os.WriteFile(toFile, []byte(content), 0644)
				if err != nil {
					return fmt.Errorf("failed to write to file %s: %w", toFile, err)
				}
				writer.Success(fmt.Sprintf("Project details written to %s successfully", toFile))
				return nil
			}

			if stdoutFlag {
				fmt.Print(content)
				return nil
			}

			return tui.RunPager(content)
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id to view. Default: active project")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump output directly to stdout without pager")
	cmd.Flags().StringVar(&toFile, "to-file", "", "Write output to the specified file path")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func updateProject(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		name        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an existing project",
		Long: `Update an existing project.
If no flags/arguments are provided, it will prompt for the details interactively.
   Examples:
     satt tracker project update <id> --name "New Name"
     satt tracker project update --name "New Name" -d "New Description"
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := ""
			if len(args) > 0 {
				pid = args[0]
			}
			if pid == "" {
				pid = cfg.CurrentProjectID()
			}
			if pid == "" {
				return fmt.Errorf("no active project configured and no projectId provided")
			}

			var (
				proj *core.Project
				err  error
			)
			_ = loader.Run("Retrieving project details...", func() {
				proj, err = service.GetProject(cmd.Context(), pid)
			})
			if err != nil {
				return err
			}

			if !cmd.Flags().Changed("name") {
				name = proj.Label
			}
			if !cmd.Flags().Changed("description") {
				description = proj.Description
			}

			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("description") {
				err = tui.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Project Name").
							Value(&name).
							Validate(func(str string) error {
								if strings.TrimSpace(str) == "" {
									return fmt.Errorf("project name is required")
								}
								return nil
							}),
						huh.NewInput().
							Title("Description").
							Value(&description),
					),
				).Run()
				if err != nil {
					return err
				}
			}

			updated, err := service.UpdateProject(cmd.Context(), pid, name, description)
			if err != nil {
				return err
			}

			writer.Success(fmt.Sprintf("Project %s updated successfully", updated.Label))
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New name of the project")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description of the project")

	return cmd
}

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
			if parentGoalID == "" && cmd.Flags().Changed("parent-goal-id") == false {
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
				sourceProjID = cfg.CurrentProjectID()
			}

			// Interactive project selection if still empty
			// TODO: This logic is probably repeated in other commands
			if sourceProjID == "" {
				projects, err := service.GetProjects(cmd.Context())
				if err != nil {
					return err
				}
				if len(projects) == 0 {
					return fmt.Errorf("no projects found")
				}

				sourceProjID, err = tui.ChooseProject(projects, "Select Source Project", "")
				if err != nil {
					return err
				}
			}

			// If goal is missing, prompt
			if splitGoalID == "" {
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

			// If target destination is missing, prompt
			if targetProjectID == "" && newProjectName == "" {
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
			if newProjectName == "" && targetParentGoalID == "" && cmd.Flags().Changed("parent") == false {
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

func deleteProjectCmd(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "delete [id]",
		Short:        "Delete an existing project and its goals",
		Aliases:      []string{"remove", "rm"},
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			err := service.DeleteProject(cmd.Context(), id)
			if err != nil {
				return err
			}
			writer.Success(fmt.Sprintf("Project %s and all its goals deleted successfully", id))

			// If we deleted the active project, clear the currentProjectId and currentGoalId
			if cfg.CurrentProjectID() == id {
				if err := cfg.SetCurrentProjectID(""); err != nil {
					writer.Warn(fmt.Sprintf("Failed to clear current project ID: %s", err))
				}
				if err := cfg.SetCurrentGoalID(""); err != nil {
					writer.Warn(fmt.Sprintf("Failed to clear current goal ID: %s", err))
				}
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
	return cmd
}
