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
				if !loader.IsTerminal() {
					return fmt.Errorf("project name is required in non-interactive mode")
				}
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
