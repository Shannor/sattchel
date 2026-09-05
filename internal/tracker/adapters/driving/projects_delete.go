package driving

import (
	"fmt"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

func deleteProjectCmd(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "delete [id]",
		Short:        "Delete an existing project and its goals",
		Aliases:      []string{"remove", "rm"},
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) > 0 {
				id = args[0]
			}
			if id == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("project ID is required in non-interactive mode")
				}
				var projects []core.Project
				var err error
				_ = loader.Run("Getting projects ...", func() {
					projects, err = service.GetProjects(cmd.Context())
				})
				if err != nil {
					return err
				}
				if len(projects) == 0 {
					return fmt.Errorf("no projects found")
				}
				selectedID, err := tui.ChooseProject(projects, "Select Project to Delete", cfg.CurrentProjectID())
				if err != nil {
					return err
				}
				id = selectedID
			}

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
