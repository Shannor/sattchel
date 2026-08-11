package driving

import (
	"fmt"
	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

func triageGoals(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
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

			// Fetch the project label once for the header
			var project *core.Project
			_ = loader.Run("Loading project info...", func() {
				project, _ = service.GetProject(cmd.Context(), pid)
			})
			projLabel := projectLabel(project, pid)

			// Determine if we should only display a specific category or filter
			hasPresetFilter := cmd.Flags().Changed("preset")
			hasMissingFilter := cmd.Flags().Changed("missing")

			// Helper function to render a goal line
			renderGoal := func(g core.Goal, isMissingCat bool) string {
				if stdoutFlag {
					if isMissingCat {
						mFields := getMissingFields(&g)
						return fmt.Sprintf("  - %s (Missing: %s) [%s]", g.Name, strings.Join(mFields, ", "), g.ID)
					}
					var parts []string
					if g.Status != "" {
						parts = append(parts, string(g.Status))
					}
					if g.Member != nil && g.Member.Name != "" {
						parts = append(parts, "@"+g.Member.Name)
					}
					line := fmt.Sprintf("  - %s [%s]", g.Name, g.ID)
					if len(parts) > 0 {
						line += " (" + strings.Join(parts, ", ") + ")"
					}
					return line
				}

				idStr := styles.Muted.Render(g.ID)
				nameStr := styles.Text.Render(g.Name)
				if isMissingCat {
					mFields := getMissingFields(&g)
					mStr := styles.Warning.Render(fmt.Sprintf("(Missing: %s)", strings.Join(mFields, ", ")))
					return fmt.Sprintf("  - %s %s [%s]", nameStr, mStr, idStr)
				}
				var meta []string
				if g.Status != "" {
					var statusStyle lipgloss.Style
					switch g.Status {
					case core.GoalCompleted:
						statusStyle = styles.Success.Bold(true)
					case core.GoalInProgress:
						statusStyle = styles.Info.Bold(true)
					case core.GoalCancelled:
						statusStyle = styles.Muted.Bold(true)
					case core.GoalOpen:
						statusStyle = styles.Info
					default:
						statusStyle = styles.Warning
					}
					meta = append(meta, statusStyle.Render(string(g.Status)))
				}
				if g.Member != nil && g.Member.Name != "" {
					meta = append(meta, styles.Info.Render("@"+g.Member.Name))
				}
				line := fmt.Sprintf("  - %s [%s]", nameStr, idStr)
				if len(meta) > 0 {
					line += "  " + strings.Join(meta, styles.Muted.Render(" • "))
				}
				return line
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
					query core.GoalQuery
					err   error
				)

				p := strings.ToLower(strings.TrimSpace(preset))
				switch p {
				case "do-it-now":
					query = core.GoalQuery{
						Impacts: []core.Impact{core.HighImpact},
						Efforts: []core.Effort{core.LowEffort},
					}
					_ = loader.Run("Getting Do It Now goals...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printTriageHeader(projLabel, styles, stdoutFlag)
					printGroup("DO IT NOW (High Impact, Low Effort)", styles.Success, goals, false)

				case "honest-work":
					query = core.GoalQuery{
						Impacts: []core.Impact{core.HighImpact},
						Efforts: []core.Effort{core.HighEffort},
					}
					_ = loader.Run("Getting Honest Work goals...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printTriageHeader(projLabel, styles, stdoutFlag)
					printGroup("HONEST WORK (High Impact, High Effort)", styles.Info, goals, false)

				case "snacking":
					query = core.GoalQuery{
						Impacts: []core.Impact{core.LowImpact},
						Efforts: []core.Effort{core.LowEffort},
					}
					_ = loader.Run("Getting Snacking goals...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printTriageHeader(projLabel, styles, stdoutFlag)
					printGroup("SNACKING (Low Impact, Low Effort)", styles.Success, goals, false)

				case "why":
					query = core.GoalQuery{
						Impacts: []core.Impact{core.LowImpact},
						Efforts: []core.Effort{core.HighEffort},
					}
					_ = loader.Run("Getting Why? goals...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printTriageHeader(projLabel, styles, stdoutFlag)
					printGroup("WHY? (Low Impact, High Effort)", styles.Warning, goals, false)

				case "missing":
					query = core.GoalQuery{
						MissingFields: []string{"member", "impact", "effort"},
					}
					var err error
					_ = loader.Run("Getting goals with missing details...", func() {
						goals, err = service.QueryGoals(cmd.Context(), pid, query)
					})
					if err != nil {
						return err
					}
					printTriageHeader(projLabel, styles, stdoutFlag)
					printGroup("MISSING IMPORTANT DETAILS", styles.Error, goals, true)

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
				_ = loader.Run("Filtering goals by missing details...", func() {
					goals, err = service.QueryGoals(cmd.Context(), pid, query)
				})
				if err != nil {
					return err
				}
				printTriageHeader(projLabel, styles, stdoutFlag)
				printGroup("GOALS MISSING SPECIFIED DETAILS", styles.Error, goals, true)
				return nil
			}

			// 3. Default: Show the complete triage summary dashboard
			var (
				allGoals []core.Goal
				err      error
			)

			_ = loader.Run("Retrieving project goals...", func() {
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

			printTriageHeader(projLabel, styles, stdoutFlag)

			printGroup("DO IT NOW (High Impact, Low Effort)", styles.Success, doItNow, false)
			printGroup("HONEST WORK (High Impact, High Effort)", styles.Info, honestWork, false)
			printGroup("SNACKING (Low Impact, Low Effort)", styles.Success, snacking, false)
			printGroup("WHY? (Low Impact, High Effort)", styles.Warning, why, false)
			printGroup("MISSING IMPORTANT DETAILS", styles.Error, missing, true)

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

func printTriageHeader(projLabel string, styles tui.Styles, stdoutFlag bool) {
	if stdoutFlag {
		fmt.Printf("%s Triage Report\n\n", projLabel)
	} else {
		fmt.Println(styles.Title.Render(projLabel + " Triage Report" + "\n"))
	}
}

func projectLabel(project *core.Project, pid string) string {
	if project != nil && project.Label != "" {
		return project.Label
	}
	return pid
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
