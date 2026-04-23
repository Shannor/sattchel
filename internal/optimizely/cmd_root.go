package optimizely

import (
	"github.com/spf13/cobra"
)

func NewCommand(s Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "optimizely",
		Short:   "Optimizely commands",
		Aliases: []string{"op"},
		Version: "0.0.1",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.AddCommand(cmdFlags(s))
	cmd.AddCommand(cmdConfig(s))
	return cmd
}
