package optcli

import (
	"sattchel/internal/optimizely"
	"sattchel/internal/printer"
	"sattchel/internal/tui"

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
