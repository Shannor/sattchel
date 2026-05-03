package update

import (
	"test-cli/internal/config"
	"test-cli/internal/printer"

	"github.com/spf13/cobra"
)

func NewCommand(writer printer.Writer) *cobra.Command {
	updater := config.NewUpdater(writer)
	return &cobra.Command{
		Use:     "update",
		Short:   "Update the CLI to the latest version",
		Aliases: []string{"u"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return updater.RunUpdate()
		},
	}
}
