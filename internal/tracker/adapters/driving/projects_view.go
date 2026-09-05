package driving

import (
	"fmt"
	"os"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

func viewProject(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		projectID  string
		stdoutFlag bool
		toFile     string
	)
	cmd := &cobra.Command{
		Use:     "view [id]",
		Aliases: []string{"v"},
		Short:   "View details of a project",
		Long: `View details of a project, including goals, status, and members.
If no projectId is provided, the current active project will be used.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := projectID
			if pid == "" && len(args) > 0 {
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

			if stdoutFlag || !loader.IsTerminal() {
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
