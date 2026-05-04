package update

import (
	"fmt"
	"test-cli/internal/config"
	"test-cli/internal/printer"

	"charm.land/huh/v2/spinner"
	"github.com/spf13/cobra"
)

func NewCommand(writer printer.Writer) *cobra.Command {
	updater := config.NewUpdater()
	return &cobra.Command{
		Use:     "update",
		Short:   "Update the CLI to the latest version",
		Aliases: []string{"u"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				update config.UpdateInformation
				err    error
			)
			if err = spinner.
				New().
				Title("Checking for updates ...").
				Action(func() {
					update, err = updater.RunUpdate()
					if err != nil {
						return
					}
				}).Run(); err != nil {
				return err
			}
			if update.NeedToUpdate {
				msg := fmt.Sprintf("Updated to %s successfully. (previous: %s)\n", update.NewVersion, update.CurrentVersion)
				writer.Success(msg)
			} else {
				writer.Info("Version is up to date\n")
			}
			return nil
		},
	}
}
