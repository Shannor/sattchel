package driving

import (
	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
)

func reopenProjectCmd(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	return setProjectStatusCmd(
		service,
		cfg,
		writer,
		"reopen",
		"Reopen a completed project",
		"Reopen a completed project by setting its status back to in-progress.",
		core.ProjectInProgress,
	)
}
