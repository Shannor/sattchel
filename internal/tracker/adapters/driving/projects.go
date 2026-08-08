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
     satt tracker project details
     satt tracker project update [id]
     `,
	}
	cmd.AddCommand(createProject(service, cfg, writer))
	cmd.AddCommand(listProjects(service, cfg, writer))
	cmd.AddCommand(projectDetails(service, cfg, writer))
	cmd.AddCommand(updateProject(service, cfg, writer))
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
				err := huh.NewForm(
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

func projectDetails(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		projectID  string
		stdoutFlag bool
		toFile     string
	)
	cmd := &cobra.Command{
		Use:     "details",
		Aliases: []string{"d"},
		Short:   "Show details of a project",
		Long: `Show details of a project, including goals, status, and members.
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

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id to get details for. Default: active project")
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
				err = huh.NewForm(
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
