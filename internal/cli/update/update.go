package update

import (
	"fmt"
	"sattchel/internal/config"
	"sattchel/internal/printer"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

var force bool

func NewCommand(writer printer.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "update",
		Short:   "Update the CLI to the latest version",
		Aliases: []string{"u"},
		RunE: func(cmd *cobra.Command, args []string) error {
			updater := config.NewUpdater()
			var update config.UpdateInformation
			var err error

			_ = loader.Run("Checking for updates ...", func() {
				update = <-updater.CheckForUpdate()
			})

			if update.NeedToUpdate || force {
				_ = loader.Run("Updating ...", func() {
					update, err = updater.RunUpdate(force)
				})
			}
			if err != nil {
				return err
			}

			if update.NeedToUpdate {
				msg := fmt.Sprintf("Updated to %s successfully (previous: %s)", update.NewVersion, update.CurrentVersion)
				writer.Success(msg)
			} else {
				writer.Info("Version is up to date")
			}
			return nil
		},
	}
	command.Flags().BoolVarP(&force, "force", "f", false, "force an update")
	return command
}
