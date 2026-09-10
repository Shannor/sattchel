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
		statuses    []string
		allStatuses bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List projects and select active project",
		Long: `List projects and select the active project.
By default, completed projects are hidden unless --all or --status is provided.

   Examples:
     satt tracker project list
     satt tracker project list --all
     satt tracker project list --status complete
     satt tracker project list --status draft --status in-progress
     satt tracker project list --set <id>
     `,
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

			statusFilterExplicit := cmd.Flags().Changed("status")
			filtered := filterProjectsForList(projects, statuses, allStatuses, statusFilterExplicit)
			if len(filtered) == 0 {
				writer.Info("No projects matched the requested filters. Use --all or --status complete to include completed projects.")
				return nil
			}

			currentProjID := cfg.CurrentProjectID()
			bypassUI := stdoutFlag || !loader.IsTerminal()
			styles := tui.AutoStyles()

			if bypassUI {
				headers := []string{"Active", "Name", "Status", "ID", "Description"}
				var rows [][]string
				for _, project := range filtered {
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
						renderProjectStatus(project, styles),
						styles.Muted.Render(project.ID),
						descVal,
					})
				}
				fmt.Fprintln(cmd.OutOrStdout(), tui.RenderTable(headers, rows))
				return nil
			}

			var options []tui.ListOption
			for _, project := range filtered {
				var titleStr string
				if project.ID == currentProjID {
					titleStr = styles.Success.Bold(true).Render("● " + project.Label + " (active)")
				} else {
					titleStr = "  " + project.Label
				}

				descStr := fmt.Sprintf("%s • %s", renderProjectStatus(project, styles), styles.Muted.Render(project.ID))
				if project.Description != "" {
					descStr += " - " + project.Description
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
				for _, p := range filtered {
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
	cmd.Flags().StringSliceVar(&statuses, "status", nil, "Filter by status (draft, in-progress, complete). Comma-separated or repeated.")
	cmd.Flags().BoolVar(&allStatuses, "all", false, "Include completed projects")
	_ = cmd.RegisterFlagCompletionFunc("set", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			string(core.ProjectDraft),
			string(core.ProjectInProgress),
			string(core.ProjectComplete),
		}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func filterProjectsForList(projects []core.Project, statuses []string, includeCompleted bool, statusFilterExplicit bool) []core.Project {
	statusSet := make(map[core.ProjectStatus]struct{}, len(statuses))
	for _, status := range statuses {
		statusSet[core.ProjectStatus(status)] = struct{}{}
	}

	var filtered []core.Project
	for _, project := range projects {
		projectStatus := project.NormalizedStatus()
		if includeCompleted {
			filtered = append(filtered, project)
			continue
		}
		if statusFilterExplicit {
			if _, ok := statusSet[projectStatus]; ok {
				filtered = append(filtered, project)
			}
			continue
		}
		if projectStatus != core.ProjectComplete {
			filtered = append(filtered, project)
		}
	}
	return filtered
}

func renderProjectStatus(project core.Project, styles tui.Styles) string {
	switch project.NormalizedStatus() {
	case core.ProjectComplete:
		return styles.Success.Bold(true).Render("Complete")
	case core.ProjectInProgress:
		return styles.Info.Bold(true).Render("In Progress")
	default:
		return styles.Warning.Render("Draft")
	}
}
