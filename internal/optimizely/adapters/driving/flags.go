package driving

import (
	"context"
	"fmt"
	"sattchel/internal/optimizely/adapters/driven"
	"sattchel/internal/optimizely/core"
	"sattchel/internal/tui"

	"charm.land/huh/v2/spinner"
	"github.com/spf13/cobra"
)

var (
	projectFilter = make([]string, 0)
	envFilter     = make([]string, 0)
	queryFilter   string
	skipCache     bool
)

func cmdFlags(s *core.Service, config *Config) *cobra.Command {
	var flagCmd = &cobra.Command{
		Use:          "flags",
		Short:        "Manage feature flags",
		Aliases:      []string{"ff"},
		SilenceUsage: true,
	}

	flagCmd.AddCommand(listFlags(s, config))
	flagCmd.AddCommand(getFlag(s, config))
	return flagCmd
}

func getFlag(s *core.Service, config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get feature flag",
		Args:  cobra.MaximumNArgs(1),
		Long: `Get feature flag.
   Examples:
     sattchel optimizely flags get <key>
     `,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("must provide a feature flag key/id")
			}
			flagId := args[0]
			ctx := cmd.Context()
			if skipCache {
				ctx = context.WithValue(ctx, driven.BypassCacheKey, true)
			}
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				return fmt.Errorf("API key is required")
			}
			projects := cfg.Projects
			if len(projects) == 0 {
				return fmt.Errorf("no projects configured")
			}
			projectID := projects[0].ID
			environments := []string{"production", "demo", "preprod", "qa", "development"}
			flag, instances, err := s.GetFlag(ctx, projectID, environments, flagId)
			if err != nil {
				return err
			}
			return tui.RenderFlagGlamour(flag, instances)
		},
	}
	cmd.Flags().StringArrayVar(&envFilter, "env", []string{}, "if provided will only show the flag for the environment(s) (if not provided will show all)")
	cmd.Flags().StringArrayVar(&projectFilter, "project", []string{}, "if provided will only show the flag for the project(s) (if not provided will show all)")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	return cmd
}

func listFlags(s *core.Service, config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List feature flags between projects",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if skipCache {
				ctx = context.WithValue(ctx, driven.BypassCacheKey, true)
			}
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				return fmt.Errorf("API key is required")
			}
			projects := cfg.Projects
			ids := make([]string, 0)
			if len(projectFilter) > 0 {
				ids = projectFilter
			} else {
				for _, project := range projects {
					ids = append(ids, project.ID)
				}
			}

			var flags map[string][]core.FeatureFlagDefinition
			if err := spinner.
				New().
				Title("Listing feature flags...").
				Action(func() {
					if queryFilter != "" {
						flags, err = s.SearchFlags(ctx, ids, core.ListFlagsOptions{Query: queryFilter})
					} else {
						flags, err = s.GetFlags(ctx, ids)
					}
				}).Run(); err != nil {
				return err
			}

			if err != nil {
				return err
			}
			for key, featureFlags := range flags {
				fmt.Printf("Project: %s - Count: %d\n", key, len(featureFlags))
				for i := range int64(4) {
					if int(i) >= len(featureFlags) {
						break
					}
					f := featureFlags[i]
					fmt.Printf("ID: %s, Name: %s\n", f.ID, f.Name)
				}
			}

			return nil
		},
	}
	cmd.Flags().StringArrayVar(&projectFilter, "filter", []string{}, "if provided will only show the flags for the provided project ids. (if not provided will show all)")
	cmd.Flags().StringArrayVar(&envFilter, "env", []string{}, "if provided will only show the flag for the environment(s) (if not provided will show all)")
	cmd.Flags().StringVar(&queryFilter, "query", "", "Filter the flags by name, key, or description substring")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	return cmd
}
