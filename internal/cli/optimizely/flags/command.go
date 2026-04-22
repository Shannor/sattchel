package flags

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "flags",
		Short:        "Manage feature flags",
		Aliases:      []string{"ff"},
		Version:      "0.0.1",
		SilenceUsage: true,
	}
	configCmd.AddCommand(add(), list(), remove())
	return configCmd
}

func add() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: ": add a feature flag",
	}
}

func list() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: ": List feature flags",
		RunE: func(cmd *cobra.Command, args []string) error {
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
