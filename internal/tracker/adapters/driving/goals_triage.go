package driving

import (
	"fmt"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

func triageGoals(service *core.Service, cfg *Config) *cobra.Command {
	var (
		projectID      string
		preset         string
		missingFilters []string
		stdoutFlag     bool
	)

	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Triage goals based on Impact/Effort presets and missing fields",
		Long: `Triage goals to prioritize them or identify missing fields.
Available Presets:
  do-it-now    - High Impact, Low Effort
  honest-work  - High Impact, High Effort
  snacking     - Low Impact, Low Effort
  why          - Low Impact, High Effort
  missing      - Goals missing member, impact, or effort

Missing Fields check can also be targeted specifically with the --missing flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := projectID
			if !cmd.Flags().Changed("projectId") {
				if lastProj := cfg.CurrentProjectID(); lastProj != "" {
					pid = lastProj
				}
			}

			if pid == "" {
				return fmt.Errorf("no project selected")
			}

			styles := tui.AutoStyles()

			// Determine if we should only display a specific category or filter
			hasPresetFilter := cmd.Flags().Changed("preset")
			hasMissingFilter := cmd.Flags().Changed("missing")

			// Helper function to render a goal line
			renderGoal := func(g core.Goal, isMissingCat bool) string {
				if stdoutFlag {
					if isMissingCat {
						mFields := getMissingFields(&g)
						return fmt.Sprintf("  - %s: %s (Missing: %s)", g.ID, g.Name, strings.Join(mFields, ", "))
					}
					return fmt.Sprintf("  - %s: %s", g.ID, g.Name)
				}

				idStr := styles.Muted.Render(g.ID)
				nameStr := styles.Text.Render(g.Name)
				if isMissingCat {
					mFields := getMissingFields(&g)
					mStr := styles.Warning.Render(fmt.Sprintf("(Missing: %s)", strings.Join(mFields, ", ")))
					return fmt.Sprintf("  - %s: %s %s", idStr, nameStr, mStr)
				}
				return fmt.Sprintf("  - %s: %s", idStr, nameStr)
			}

			// Helper function to print a group of goals
			printGroup := func(header string, headerStyle lipgloss.Style, list []core.Goal, isMissingCat bool) {
				if len(list) == 0 {
					return
				}
				if stdoutFlag {
					fmt.Println(header)
				} else {
					fmt.Println(headerStyle.Bold(true).Render(header))
				}
				for _, g := range list {
					fmt.Println(renderGoal(g, isMissingCat))
				}
				fmt.Println()
			}

			// 1. Specific preset filtering requested
			if hasPresetFilter {
				var (
					goals []core.Goal
					err   error
					query core.GoalQuery
				)

				p := strings.ToLower(strings.TrimSpace(preset))
				switch p {
				case "do-it-now":
					query = core.GoalQuery{
						Impacts: []core.Impact{core.HighImpact},
						Efforts: []core.Effort{core.LowEffort},
					}
					err = loader.Run("Getting Do It Now goals...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printGroup("🚀 DO IT NOW (High Impact, Low Effort)", styles.Success, goals, false)

				case "honest-work":
					query = core.GoalQuery{
						Impacts: []core.Impact{core.HighImpact},
						Efforts: []core.Effort{core.HighEffort},
					}
					err = loader.Run("Getting Honest Work goals...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printGroup("🛠️ HONEST WORK (High Impact, High Effort)", styles.Info, goals, false)

				case "snacking":
					query = core.GoalQuery{
						Impacts: []core.Impact{core.LowImpact},
						Efforts: []core.Effort{core.LowEffort},
					}
					err = loader.Run("Getting Snacking goals...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printGroup("🍿 SNACKING (Low Impact, Low Effort)", styles.Success, goals, false)

				case "why":
					query = core.GoalQuery{
						Impacts: []core.Impact{core.LowImpact},
						Efforts: []core.Effort{core.HighEffort},
					}
					err = loader.Run("Getting Why? goals...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printGroup("⚠️ WHY? (Low Impact, High Effort)", styles.Warning, goals, false)

				case "missing":
					query = core.GoalQuery{
						MissingFields: []string{"member", "impact", "effort"},
					}
					err = loader.Run("Getting goals with missing details...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printGroup("🔧 MISSING IMPORTANT DETAILS", styles.Error, goals, true)

				default:
					return fmt.Errorf("unknown preset: %s (supported: do-it-now, honest-work, snacking, why, missing)", preset)
				}

				return nil
			}

			// 2. Specific missing fields filtering requested
			if hasMissingFilter {
				var (
					goals []core.Goal
					err   error
				)

				query := core.GoalQuery{
					MissingFields: missingFilters,
				}
				err = loader.Run("Filtering goals by missing details...", func() {
					goals, err = service.QueryGoals(cmd.Context(), pid, query)
				})
				if err != nil {
					return err
				}
				printGroup("🔧 GOALS MISSING SPECIFIED DETAILS", styles.Error, goals, true)
				return nil
			}

			// 3. Default: Show the complete triage summary dashboard
			var (
				allGoals []core.Goal
				err      error
			)

			err = loader.Run("Retrieving project goals...", func() {
				allGoals, err = service.QueryGoals(cmd.Context(), pid, core.GoalQuery{})
			})
			if err != nil {
				return err
			}

			if len(allGoals) == 0 {
				fmt.Printf("No goals found for project %s\n", pid)
				return nil
			}

			// Categorize full list for display
			var (
				doItNow    []core.Goal
				honestWork []core.Goal
				snacking   []core.Goal
				why        []core.Goal
				missing    []core.Goal
			)

			for _, g := range allGoals {
				if g.IsDoItNow() {
					doItNow = append(doItNow, g)
				} else if g.IsHonestWork() {
					honestWork = append(honestWork, g)
				} else if g.IsSnacking() {
					snacking = append(snacking, g)
				} else if g.IsWhy() {
					why = append(why, g)
				}

				mFields := getMissingFields(&g)
				if len(mFields) > 0 {
					missing = append(missing, g)
				}
			}

			if stdoutFlag {
				fmt.Println("==== Triage Dashboard ====")
				fmt.Println()
			} else {
				dashboardTitle := styles.Title.Render(" ==== TRIAGE DASHBOARD ==== ")
				fmt.Println("\n" + dashboardTitle + "\n")
			}

			printGroup("🚀 DO IT NOW (High Impact, Low Effort)", styles.Success, doItNow, false)
			printGroup("🛠️ HONEST WORK (High Impact, High Effort)", styles.Info, honestWork, false)
			printGroup("🍿 SNACKING (Low Impact, Low Effort)", styles.Success, snacking, false)
			printGroup("⚠️ WHY? (Low Impact, High Effort)", styles.Warning, why, false)
			printGroup("🔧 MISSING IMPORTANT DETAILS", styles.Error, missing, true)

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project ID of the goals. If not provided, the default project will be used")
	cmd.Flags().StringVarP(&preset, "preset", "r", "", "Filter goals by priority preset (do-it-now, honest-work, snacking, why, missing)")
	cmd.Flags().StringSliceVarP(&missingFilters, "missing", "m", []string{}, "Filter goals missing specific fields (description, member, impact, effort)")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Output plain text instead of formatted CLI styles")

	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("preset", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		presets := []string{"do-it-now", "honest-work", "snacking", "why", "missing"}
		return presets, cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.RegisterFlagCompletionFunc("missing", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		fields := []string{"description", "member", "impact", "effort"}
		return fields, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func getMissingFields(g *core.Goal) []string {
	var missing []string
	if g.Member == nil || g.Member.ID == "" {
		missing = append(missing, "member")
	}
	if g.Impact == core.UnknownImpact || g.Impact == "" {
		missing = append(missing, "impact")
	}
	if g.Effort == core.UnknownEffort || g.Effort == "" {
		missing = append(missing, "effort")
	}
	return missing
}
