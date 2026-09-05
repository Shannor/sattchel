package driving

import (
	"fmt"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"
	"sattchel/pkg/set"

	"github.com/spf13/cobra"
)

func moveGoal(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var (
		relationship = core.LinkOptional
		projectID    string
	)

	cmd := &cobra.Command{
		Use:     "move [childId] [newParentId]",
		Short:   "Move a goal to a new parent",
		Aliases: []string{"mv"},
		Long: `Move a goal to a new parent.
   If childId and newParentId are not provided, an interactive prompt will be displayed.
   Examples:
     satt tracker goals move
     satt tracker goals move <childId> <newParentId> -r <relationship>
     `,
		Args:         cobra.MaximumNArgs(2),
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			pid := getActiveProjectID(cmd, cfg, projectID)
			if pid == "" {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			goals, err := service.GetGoals(cmd.Context(), pid)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 0 {
				var completions []string
				for _, g := range goals {
					if g.IsRoot() {
						continue
					}
					completions = append(completions, cobra.CompletionWithDesc(g.ID, g.Name))
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			if len(args) == 1 {
				childID := args[0]
				allowedParents, err := service.GetAllowedParents(cmd.Context(), pid, childID)
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}

				allowedSet := set.NewFromFunc(allowedParents, func(g core.Goal) string { return g.ID })
				var completions []string
				for _, g := range goals {
					if !allowedSet.Contains(g.ID) {
						continue
					}
					completions = append(completions, cobra.CompletionWithDesc(g.ID, g.Name))
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}

			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := ensureProjectID(cmd, service, cfg, projectID)
			if err != nil {
				return err
			}

			var childID string
			var newParentID string

			if len(args) >= 1 {
				childID = args[0]
			}
			if len(args) == 2 {
				newParentID = args[1]
			}

			if (childID == "" || newParentID == "") && !loader.IsTerminal() {
				return fmt.Errorf("both childId and newParentId are required in non-interactive mode")
			}

			var (
				goals []core.Goal
			)
			_ = loader.Run("Getting goals ...", func() {
				goals, err = service.GetGoals(cmd.Context(), pid)
			})
			if err != nil {
				return err
			}
			if len(goals) == 0 {
				return fmt.Errorf("no goals found for project %s", pid)
			}

			rootGoal, err := service.GetRootGoal(cmd.Context(), pid)
			if err != nil {
				return err
			}

			if childID != "" && childID == rootGoal.ID {
				return fmt.Errorf("the root goal cannot be moved")
			}

			if childID == "" {
				currentGoalID := cfg.CurrentGoalID()
				childID, err = tui.ChooseGoal(goals, "Select Goal to Move", currentGoalID, nil, func(val string) error {
					if val == rootGoal.ID {
						return fmt.Errorf("the root goal cannot be moved")
					}
					return nil
				})
				if err != nil {
					return err
				}
			}

			allowedGoals, err := service.GetAllowedParents(cmd.Context(), pid, childID)
			if err != nil {
				return err
			}
			if len(allowedGoals) == 0 {
				return fmt.Errorf("no valid parent goals available to move this goal under")
			}
			allowedSet := set.NewFromFunc(allowedGoals, func(g core.Goal) string { return g.ID })
			if newParentID == "" {
				newParentID, err = tui.ChooseGoal(goals, "Select New Parent Goal", "", func(g *core.Goal) bool {
					return allowedSet.Contains(g.ID)
				}, nil)
				if err != nil {
					return err
				}
			}

			if !cmd.Flags().Changed("relationship") && len(args) == 0 && loader.IsTerminal() {
				relOptions := []tui.ListOption{
					{TitleStr: "Optional", DescriptionStr: "Child goal is optional", ValueStr: string(core.LinkOptional)},
					{TitleStr: "Required", DescriptionStr: "Child goal is required before parent completion", ValueStr: string(core.LinkRequired)},
				}
				selectedRel, err := tui.Choose("Select Link Relationship", relOptions)
				if err != nil {
					return err
				}
				if selectedRel == nil {
					return fmt.Errorf("no relationship selected")
				}
				relationship = core.LinkRelationship(selectedRel.ValueStr)
			}

			var movedGoal *core.Goal
			_ = loader.Run("Moving goal ...", func() {
				movedGoal, err = service.ChangeParent(cmd.Context(), pid, childID, newParentID, core.GoalOptions{
					LinkRelationship: relationship,
				})
			})
			if err != nil {
				return err
			}

			writer.Success(fmt.Sprintf("Goal %q (%s) moved successfully under parent %s", movedGoal.Name, movedGoal.ID, newParentID))
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goal. If not provided, the default project will be used")
	cmd.Flags().StringVarP((*string)(&relationship), "relationship", "r", string(core.LinkOptional), "Relationship of the link between the goal and its new parent")
	_ = cmd.RegisterFlagCompletionFunc("relationship", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{string(core.LinkOptional), string(core.LinkRequired)}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}
