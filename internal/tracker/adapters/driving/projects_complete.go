package driving

import (
	"fmt"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

func completeProjectCmd(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	return setProjectStatusCmd(
		service,
		cfg,
		writer,
		"complete",
		"Mark a project as complete",
		"Mark a project as complete. Completed projects are hidden from project list by default.",
		core.ProjectComplete,
	)
}

func setProjectStatusCmd(service *core.Service, cfg *Config, writer printer.Writer, use string, short string, long string, status core.ProjectStatus) *cobra.Command {
	cmd := &cobra.Command{
		Use:          use + " [id]",
		Short:        short,
		Long:         long,
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
				project *core.Project
				err     error
			)
			_ = loader.Run("Updating project status...", func() {
				project, err = service.UpdateProject(cmd.Context(), pid, "", "", status)
			})
			if err != nil {
				return err
			}

			writer.Success(fmt.Sprintf("Project %s marked as %s", project.Label, project.Status))
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
		},
	}

	return cmd
}
