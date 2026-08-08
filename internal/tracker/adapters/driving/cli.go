package driving

import (
	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCommand(service *core.Service, v *viper.Viper, writer printer.Writer) *cobra.Command {
	cfg, _ := LoadConfig(v)
	cmd := &cobra.Command{
		Use:     "tracker",
		Short:   "Tracker Commands",
		Aliases: []string{"tr"},
	}
	cmd.AddCommand(projects(service, cfg, writer))
	cmd.AddCommand(goals(service, cfg))
	cmd.AddCommand(members(service, cfg))
	cmd.AddCommand(visualizeProject(service, cfg))
	cmd.AddCommand(exportCmd(service))
	cmd.AddCommand(importCmd(service))
	return cmd
}
