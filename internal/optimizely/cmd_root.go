package optimizely

import (
	"test-cli/internal/printer"

	"github.com/spf13/cobra"
)

func NewCommand(s Service, styles printer.Writer) *cobra.Command {
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
	cmd.AddCommand(cmdProjects(s, styles))
	cmd.AddCommand(cmdConfig(s))
	return cmd
}
