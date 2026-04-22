package optimizely

import (
	"test-cli/internal/cli"
	"test-cli/internal/cli/optimizely/flags"
	"test-cli/internal/config"

	"github.com/spf13/cobra"
)

func NewCommand(deps cli.Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "optimizely",
		Short:   "Optimizely commands",
		Aliases: []string{"op"},
		Version: "0.0.1",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			svc := config.NewConfigurationService()
			err := svc.Init()
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(flags.NewCommand())
	return cmd
}
