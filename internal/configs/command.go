package configs

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {

	// configCmd represents the config command
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Mange CLI configs for Optimizely, Contentful, and more",
	}
	configCmd.AddCommand(set())
	configCmd.AddCommand(get())
	configCmd.AddCommand(all())
	// TODO: add subcommands for delete, and more
	return configCmd
}
