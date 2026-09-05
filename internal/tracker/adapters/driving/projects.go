package driving

import (
	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
)

func projects(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project [next]",
		Aliases: []string{"p"},
		Short:   "Manage projects",
		Long: `Manage projects.
   Examples:
     satt tracker project create <name> 
     satt tracker project list
     satt tracker project view
     satt tracker project update [id]
     satt tracker project merge <source_project_id> <merge_project_id>
     satt tracker project split <source_project_id>
     `,
	}
	cmd.AddCommand(createProject(service, cfg, writer))
	cmd.AddCommand(listProjects(service, cfg, writer))
	cmd.AddCommand(viewProject(service, cfg, writer))
	cmd.AddCommand(updateProject(service, cfg, writer))
	cmd.AddCommand(mergeProjectsCmd(service, cfg, writer))
	cmd.AddCommand(splitProjectCmd(service, cfg, writer))
	cmd.AddCommand(deleteProjectCmd(service, cfg, writer))
	return cmd
}
