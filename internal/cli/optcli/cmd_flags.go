package optcli

import (
	"fmt"
	"test-cli/internal/models"
	"test-cli/internal/optimizely"
	"test-cli/internal/tui"

	"github.com/spf13/cobra"
)

var (
	projectFilter = make([]string, 0)
)

func cmdFlags(s optimizely.Service) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "flags",
		Short:        "Manage feature flags",
		Aliases:      []string{"ff"},
		SilenceUsage: true,
	}

	configCmd.AddCommand(compare(s))
	configCmd.AddCommand(list(s))
	return configCmd
}

// TODO: Would need to see our existing Compare and use it as a guide
// Should need a flags filter, brands filter, etc.

func compare(s optimizely.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare feature flags between projects",
		RunE: func(cmd *cobra.Command, args []string) error {

			return nil
		},
	}
	return cmd
}

func list(s optimizely.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List feature flags between projects",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, err := s.GetConfig()
			if err != nil {
				return err
			}
			projects := cfg.Projects
			ids := make([]string, 0)
			for _, project := range projects {
				ids = append(ids, project.ID)
			}

			spinner := tui.NewSpinner()
			spinner.Start()
			defer spinner.Stop()

			reporter := &tui.TerminalReporter{
				Spinner: spinner,
			}

			ctx = models.WithProgress(ctx, reporter)
			flags, err := s.GetFlags(ctx, ids)
			if err != nil {
				return err
			}
			for key, featureFlags := range flags {
				fmt.Printf("Project: %s - Count: %d\n", key, len(featureFlags))
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&projectFilter, "filter", []string{}, "if provided will only show the flags for the provided project ids. (if not provided will show all)")
	return cmd
}
