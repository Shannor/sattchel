package driving

import (
	"fmt"
	"sattchel/internal/optimizely/core"
	"sattchel/internal/tui"

	"github.com/spf13/cobra"
)

var (
	projectFilter = make([]string, 0)
	envFilter     = make([]string, 0)
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
	return cmd
}

func listFlags(s *core.Service, config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List feature flags between projects",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				return fmt.Errorf("API key is required")
			}
			projects := cfg.Projects
			ids := make([]string, 0)
			for _, project := range projects {
				ids = append(ids, project.ID)
				break
			}

			spinner := tui.NewSpinner()
			spinner.Start()
			defer spinner.Stop()

			reporter := &tui.TerminalReporter{
				Spinner: spinner,
			}

			ctx = core.WithProgress(ctx, reporter)
			flags, err := s.GetFlags(ctx, ids)
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
	return cmd
}
