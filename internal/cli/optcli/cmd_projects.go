package optcli

import (
	"fmt"
	"sattchel/internal/domain"
	"sattchel/internal/optimizely"
	"sattchel/internal/printer"
	"slices"
	"strings"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/spf13/cobra"
)

// TODO: Need a way to edit Project names, so they are easily identifiable

func cmdProjects(s optimizely.Service, writer printer.Writer) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "projects",
		Short:        "Manage projects",
		SilenceUsage: true,
	}

	configCmd.AddCommand(listProjects(s, writer))
	return configCmd
}

func listProjects(s optimizely.Service, writer printer.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Projects",
		RunE: func(cmd *cobra.Command, args []string) error {

			var (
				projects []domain.Project
				err      error
			)

			if err := spinner.
				New().
				Title("Getting projects ...").
				Action(func() {
					projects, err = s.GetAllProjects(cmd.Context())
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

			slices.SortFunc(projects, func(i, j domain.Project) int {
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
			var results []domain.Project
			for _, project := range projects {
				if selectedSet[project.ID] {
					results = append(results, project)
				}
			}

			c := optimizely.Configuration{Projects: results}
			err = s.SetConfig(cmd.Context(), c)
			if err != nil {
				return err
			}

			writer.Success("Set configuration successfully")
			return nil
		},
	}
}
