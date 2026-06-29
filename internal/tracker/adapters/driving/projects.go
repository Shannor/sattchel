package driving

import (
	"fmt"
	"sattchel/internal/tracker/core"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/spf13/cobra"
)

func projects(service *core.Service, cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project [next]",
		Aliases: []string{"p"},
		Short:   "Manage projects",
		Long: `Manage projects.
   Examples:
     sattchel tracker project create <name> 
     sattchel tracker project set
     sattchel tracker project list
     `,
	}
	cmd.AddCommand(createProject(service, cfg))
	cmd.AddCommand(setProject(service, cfg))
	cmd.AddCommand(listProjects(service, cfg))
	return cmd
}

func createProject(service *core.Service, cfg *Config) *cobra.Command {
	description := ""
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Long: `Create a new project.
	If no key is provided, a list of available keys will be displayed.
   Examples:
     sattchel tracker project create <name> 
     sattchel tracker project create <name> -d "description"
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("project name is required")
			}
			p, err := service.CreateProject(cmd.Context(), args[0], description)
			if err != nil {
				return err
			}
			fmt.Printf("Project %s created successfully\n", p.Label)
			_ = cfg.SetCurrentProjectID(p.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "description of the project")
	return cmd
}

func setProject(service *core.Service, cfg *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "set",
		Short: "Set Active Project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				projects []core.Project
				err      error
			)

			if err := spinner.
				New().
				Title("Getting projects ...").
				Action(func() {
					projects, err = service.GetProjects(cmd.Context())
				}).Run(); err != nil {
				return err
			}

			if err != nil {
				return err
			}

			if len(projects) == 0 {
				return fmt.Errorf("no projects found")
			}

			var (
				selectedID string
				options    []huh.Option[string]
			)

			currentProjID := cfg.CurrentProjectID()
			for _, project := range projects {
				option := huh.NewOption(project.Label, project.ID)
				if project.ID == currentProjID {
					option = option.Selected(true)
					selectedID = project.ID
				}
				options = append(options, option)
			}

			err = huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select Active Project").
						Options(options...).
						Value(&selectedID),
				),
			).WithShowHelp(true).Run()
			if err != nil {
				return err
			}

			if err := cfg.SetCurrentProjectID(selectedID); err != nil {
				return fmt.Errorf("failed to save active project: %w", err)
			}

			var name string
			for _, p := range projects {
				if p.ID == selectedID {
					name = p.Label
					break
				}
			}

			fmt.Printf("Active project set to: %s (%s)\n", name, selectedID)
			return nil
		},
	}
}

func listProjects(service *core.Service, cfg *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				projects []core.Project
				err      error
			)

			if err := spinner.
				New().
				Title("Getting projects ...").
				Action(func() {
					projects, err = service.GetProjects(cmd.Context())
				}).Run(); err != nil {
				return err
			}

			if err != nil {
				return err
			}

			if len(projects) == 0 {
				fmt.Println("No projects found")
				return nil
			}

			currentProjID := cfg.CurrentProjectID()
			for _, project := range projects {
				currentMarker := ""
				if project.ID == currentProjID {
					currentMarker = " (active)"
				}
				fmt.Printf("- %s: %s%s\n", project.Label, project.ID, currentMarker)
			}
			return nil
		},
	}
}
