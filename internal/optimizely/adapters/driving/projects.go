package driving

import (
	"context"
	"fmt"
	"sattchel/internal/optimizely/core"
	"sattchel/internal/printer"
	"sattchel/pkg/set"
	"slices"
	"strings"

	"sattchel/pkg/loader"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func cmdProjects(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "projects",
		Short:        "Manage projects",
		SilenceUsage: true,
	}

	configCmd.AddCommand(setProjects(s, config, writer))
	configCmd.AddCommand(listProjects(s, config, writer))
	return configCmd
}

func getAllProjects(ctx context.Context, s *core.Service, config *Config) ([]core.Project, error) {
	cfg, err := config.Get()
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	flagsProjects, err := s.GetProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	existing := set.NewFromFunc[string](cfg.Projects, func(v core.Project) string {
		return v.ID
	})

	var results []core.Project
	for _, proj := range flagsProjects {
		id := proj.ID
		results = append(results, core.Project{
			ID:       id,
			Name:     proj.Name,
			IsActive: existing.Contains(id),
			Label:    proj.Label,
		})
	}
	return results, nil
}

func setProjects(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "set",
		Short: "Pick projects",
		RunE: func(cmd *cobra.Command, args []string) error {

			var (
				projects []core.Project
				err      error
			)

			err = loader.Run("Getting projects ...", func() {
				projects, err = getAllProjects(cmd.Context(), s, config)
			})
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

			slices.SortFunc(projects, func(i, j core.Project) int {
				return strings.Compare(i.Name, j.Name)
			})

			for _, project := range projects {
				options = append(options, huh.NewOption(project.Name, project.ID).Selected(project.IsActive))
				if project.IsActive {
					selectedIds = append(selectedIds, project.ID)
				}
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
			var results []core.Project
			for _, project := range projects {
				if selectedSet[project.ID] {
					results = append(results, project)
				}
			}

			_, err = config.Update(func(cfg *Configuration) error {
				cfg.Projects = results
				return nil
			})
			if err != nil {
				return err
			}

			writer.Success("Set configuration successfully")
			return nil
		},
	}
}

func listProjects(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			var projects []core.Project
			var err error
			err = loader.Run("Getting projects ...", func() {
				projects, err = getAllProjects(cmd.Context(), s, config)
			})
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				writer.Info("No projects found")
				return nil
			}

			slices.SortFunc(projects, func(i, j core.Project) int {
				return strings.Compare(i.Name, j.Name)
			})

			for _, project := range projects {
				activeMarker := ""
				if project.IsActive {
					activeMarker = " (active)"
				}
				fmt.Printf("- %s: %s%s\n", project.Name, project.ID, activeMarker)
			}
			return nil
		},
	}
}
