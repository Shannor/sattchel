package driving

import (
	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
)

func goals(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "goals [verb]",
		Short:   "Manage goals",
		Aliases: []string{"g"},
		Long: `Commands to manage goals.
     Examples:
       satt tracker goals add <name>
       satt tracker goals set
       satt tracker goals list
       satt tracker goals move <childId> <newParentId>
       satt tracker goals update <id>
       satt tracker goals delete <id>
       satt tracker goals view <id>
       satt tracker goals triage
       `,
	}
	cmd.AddCommand(addGoal(service, cfg, writer))
	cmd.AddCommand(setGoal(service, cfg, writer))
	cmd.AddCommand(listGoals(service, cfg, writer))
	cmd.AddCommand(moveGoal(service, cfg, writer))
	cmd.AddCommand(deleteGoal(service, cfg, writer))
	cmd.AddCommand(viewGoal(service, cfg, writer))
	cmd.AddCommand(updateGoal(service, cfg, writer))
	cmd.AddCommand(triageGoals(service, cfg, writer))
	return cmd
}
