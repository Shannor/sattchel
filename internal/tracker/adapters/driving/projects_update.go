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
				var err error
				pid, err = ensureProjectID(cmd, service, cfg, "")
				if err != nil {
					return err
				}
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
				if !loader.IsTerminal() {
					return fmt.Errorf("at least one flag (--name or --description) must be specified for update in non-interactive mode")
				}
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
