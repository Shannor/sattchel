package driving

import (
	"fmt"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

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
