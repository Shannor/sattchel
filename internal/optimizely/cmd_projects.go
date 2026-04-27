package optimizely

import (
	"fmt"
	"slices"
	"strings"
	"test-cli/internal/tui"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/spf13/cobra"
)

// TODO: Need a way to edit Project names, so they are easily identifiable

func cmdProjects(s Service, styles tui.Styles) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "projects",
		Short:        "Manage projects",
		Version:      "0.0.1",
		SilenceUsage: true,
	}

	configCmd.AddCommand(listProjects(s, styles))
	return configCmd
}

func listProjects(s Service, styles tui.Styles) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Projects",
		RunE: func(cmd *cobra.Command, args []string) error {

			var (
				projects []Project
				err      error
			)

			if err := spinner.
				New().
				Title("Getting projects ...").
				Action(func() {
					projects, err = s.GetProjects(cmd.Context())
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
				selectedIds []string
				options     []huh.Option[string]
			)

			slices.SortFunc(projects, func(i, j Project) int {
				return strings.Compare(i.Name, j.Name)
			})

			for _, project := range projects {
				options = append(options, huh.NewOption(project.Name, project.ID).Selected(project.IsActive))
			}

			err = huh.NewForm(
				huh.NewGroup(
					huh.NewMultiSelect[string]().
						Title("Available Projects").
						Options(options...).
						Value(&selectedIds),
				),
			).WithShowHelp(true).Run()
			if err != nil {
				return err
			}
			selectedSet := make(map[string]bool)
			for _, id := range selectedIds {
				selectedSet[id] = true
			}
			var results []Project
			for _, project := range projects {
				if selectedSet[project.ID] {
					results = append(results, project)
				}
			}

			c := Configuration{Projects: results}
			err = s.SetConfig(c)
			if err != nil {
				return err
			}
			fmt.Println(styles.Success.Render("Set configuration successfully"))
			return nil
		},
	}
}
