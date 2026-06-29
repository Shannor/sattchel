package driving

import (
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
)

func NewCommand(service *core.Service) *cobra.Command {
	cfg, _ := LoadConfig()
	cmd := &cobra.Command{
		Use:     "tracker",
		Short:   "Tracker Commands",
		Aliases: []string{"tr"},
	}
	cmd.AddCommand(projects(service, cfg))
	cmd.AddCommand(goals(service, cfg))
	return cmd
}
