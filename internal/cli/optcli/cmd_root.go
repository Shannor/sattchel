package optcli

import (
	"test-cli/internal/optimizely"
	"test-cli/internal/printer"
	"test-cli/internal/tui"

	"github.com/spf13/cobra"
)

func NewCommand(s optimizely.Service, writer printer.Writer, styles tui.Styles) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "optimizely",
		Short:   "Optimizely commands",
		Aliases: []string{"op"},
	}
	cmd.AddCommand(cmdFlags(s))
	cmd.AddCommand(cmdProjects(s, writer))
	cmd.AddCommand(cmdConfig(s, styles))
	return cmd
}
