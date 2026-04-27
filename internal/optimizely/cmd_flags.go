package optimizely

import (
	"github.com/spf13/cobra"
)

func cmdFlags(s Service) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "flags",
		Short:        "Manage feature flags",
		Aliases:      []string{"ff"},
		Version:      "0.0.1",
		SilenceUsage: true,
	}

	configCmd.AddCommand(compare(s))
	return configCmd
}

// TODO: Would need to see our existing Compare and use it as a guide
// Should need a flags filter, brands filter, etc.

func compare(s Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare feature flags between projects",
		RunE: func(cmd *cobra.Command, args []string) error {

			return nil
		},
	}
	return cmd
}
