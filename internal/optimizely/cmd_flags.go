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

	configCmd.AddCommand(add(), list(), remove(), compare())
	return configCmd
}

func add() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: ": add a feature flag",
	}
}

func compare() *cobra.Command {
	return &cobra.Command{
		Use:   "compare",
		Short: "Comapre feature flags between projects",
	}
}
func list() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List feature flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Steps needed to list flags
			return nil
		},
	}
}

func remove() *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: ": Remove feature flag",
	}
}
