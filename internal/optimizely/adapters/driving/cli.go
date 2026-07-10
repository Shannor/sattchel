package driving

import (
	"sattchel/internal/optimizely/core"
	"sattchel/internal/printer"
	"sattchel/internal/tui"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCommand(s *core.Service, v *viper.Viper, writer printer.Writer, styles tui.Styles) *cobra.Command {
	cfg := NewConfig(v)
	cmd := &cobra.Command{
		Use:     "optimizely",
		Short:   "Optimizely commands",
		Aliases: []string{"op"},
	}
	cmd.AddCommand(cmdFlags(s, cfg, writer))
	cmd.AddCommand(cmdProjects(s, cfg, writer))
	cmd.AddCommand(cmdConfig(cfg, styles))
	cmd.AddCommand(cmdCache(cfg))
	return cmd
}
