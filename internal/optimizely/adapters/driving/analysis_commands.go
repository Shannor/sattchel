package driving

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sattchel/internal/optimizely/adapters/driven"
	"sattchel/internal/optimizely/core"
	"sattchel/internal/printer"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func uniqueFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var (
		targetProjectID string
		againstProjects []string
		query           string
		skipCacheFlag   bool
		stdout          bool
		toFilePath      string
		jsonOutput      bool
	)

	cmd := &cobra.Command{
		Use:   "unique",
		Short: "List flags unique to one project",
		Long:  "Find flags that exist only in one project and are absent from the other selected projects.",
		Args:  cobra.NoArgs,
		Example: strings.TrimSpace(`
  satt optimizely flags unique --project 123
  satt optimizely flags unique --project 123 --against 456 --against 789
  satt optimizely flags unique --project 123 --query loyalty --stdout
`),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cfg, err := analysisContextAndConfig(cmd, config, skipCacheFlag)
			if err != nil {
				return err
			}
			if targetProjectID == "" {
				return fmt.Errorf("--project is required")
			}
			if len(againstProjects) == 0 {
				againstProjects = projectsExcept(configuredProjectIDs(cfg), targetProjectID)
			}
			if len(againstProjects) == 0 {
				return fmt.Errorf("no comparison projects available")
			}

			var result []core.UniqueFlagEntry
			err = loader.Run("Finding unique flags...", func() {
				result, err = s.FindUniqueFlags(ctx, targetProjectID, againstProjects, query)
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				return writeJSONOutput(result, toFilePath, stdout)
			}
			return writeAnalysisOutput(tui.RenderOptimizelyUniqueFlags(result), toFilePath, stdout)
		},
	}

	cmd.Flags().StringVarP(&targetProjectID, "project", "p", "", "Project ID to check uniqueness for")
	cmd.Flags().StringSliceVar(&againstProjects, "against", nil, "Project IDs to compare against (defaults to other configured projects)")
	cmd.Flags().StringVar(&query, "query", "", "Filter flags by name, key, or description substring")
	cmd.Flags().BoolVar(&skipCacheFlag, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Dump output directly to stdout without pager")
	cmd.Flags().StringVar(&toFilePath, "to-file", "", "Write output to the specified file path")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write JSON output")
	registerProjectFlagCompletion(cmd, config, "project", "against")
	_ = cmd.MarkFlagRequired("project")
	return cmd
}

func dormantFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var (
		projectIDs    []string
		query         string
		skipCacheFlag bool
		stdout        bool
		toFilePath    string
		jsonOutput    bool
	)

	cmd := &cobra.Command{
		Use:   "dormant",
		Short: "List flags disabled in every environment across the selected projects",
		Long:  "Find flags that are disabled everywhere across all selected projects that contain them.",
		Args:  cobra.NoArgs,
		Example: strings.TrimSpace(`
  satt optimizely flags dormant
  satt optimizely flags dormant --project 123 --project 456 --json
`),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cfg, err := analysisContextAndConfig(cmd, config, skipCacheFlag)
			if err != nil {
				return err
			}
			targetProjectIDs := projectIDs
			if len(targetProjectIDs) == 0 {
				targetProjectIDs = configuredProjectIDs(cfg)
			}
			if len(targetProjectIDs) == 0 {
				return fmt.Errorf("at least 1 project ID is required")
			}

			var result []core.DormantFlagEntry
			err = loader.Run("Finding dormant flags...", func() {
				result, err = s.FindDormantFlags(ctx, targetProjectIDs, query)
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				return writeJSONOutput(result, toFilePath, stdout)
			}
			return writeAnalysisOutput(tui.RenderOptimizelyDormantFlags(result, targetProjectIDs), toFilePath, stdout)
		},
	}

	cmd.Flags().StringSliceVarP(&projectIDs, "project", "p", nil, "Project IDs to inspect (defaults to configured projects)")
	cmd.Flags().StringVar(&query, "query", "", "Filter flags by name, key, or description substring")
	cmd.Flags().BoolVar(&skipCacheFlag, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Dump output directly to stdout without pager")
	cmd.Flags().StringVar(&toFilePath, "to-file", "", "Write output to the specified file path")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write JSON output")
	registerProjectFlagCompletion(cmd, config, "project")
	return cmd
}

func driftFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var (
		projectIDs    []string
		query         string
		skipCacheFlag bool
		stdout        bool
		toFilePath    string
		jsonOutput    bool
	)

	cmd := &cobra.Command{
		Use:   "drift",
		Short: "List shared flags that are missing variable definitions in some projects",
		Long:  "Detect variable definitions that are missing from some projects for same-key flags.",
		Args:  cobra.NoArgs,
		Example: strings.TrimSpace(`
  satt optimizely flags drift
  satt optimizely flags drift --project 123 --project 456 --query checkout
  satt optimizely flags drift --json
`),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cfg, err := analysisContextAndConfig(cmd, config, skipCacheFlag)
			if err != nil {
				return err
			}
			targetProjectIDs := projectIDs
			if len(targetProjectIDs) == 0 {
				targetProjectIDs = configuredProjectIDs(cfg)
			}
			if len(targetProjectIDs) < 2 {
				return fmt.Errorf("at least 2 project IDs are required")
			}

			var result []core.FlagVariableDrift
			err = loader.Run("Finding variable drift...", func() {
				result, err = s.FindVariableDrift(ctx, targetProjectIDs, query)
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				return writeJSONOutput(result, toFilePath, stdout)
			}
			return writeAnalysisOutput(tui.RenderOptimizelyVariableDrift(result), toFilePath, stdout)
		},
	}

	cmd.Flags().StringSliceVarP(&projectIDs, "project", "p", nil, "Project IDs to inspect (defaults to configured projects)")
	cmd.Flags().StringVar(&query, "query", "", "Filter flags by name, key, or description substring")
	cmd.Flags().BoolVar(&skipCacheFlag, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Dump output directly to stdout without pager")
	cmd.Flags().StringVar(&toFilePath, "to-file", "", "Write output to the specified file path")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write JSON output")
	registerProjectFlagCompletion(cmd, config, "project")
	return cmd
}

func syncFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var (
		sourceProjectID  string
		targetProjectIDs []string
		flagKeys         []string
		updateVars       bool
		apply            bool
		yes              bool
		skipCacheFlag    bool
		stdout           bool
		toFilePath       string
		jsonOutput       bool
		syncVariations   bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Plan or apply missing flag and variable syncs across projects",
		Long:  "Plan or apply synchronization of missing flags and variable definitions from one project to others, or in union mode across all targets.",
		Example: strings.TrimSpace(`
  satt optimizely flags sync --source 123 --target 456
  satt optimizely flags sync --source 123 --target 456 --update-vars
  satt optimizely flags sync --source all --target 123 --target 456
  satt optimizely flags sync --source 123 --target 456 --flag loyalty_checkout --apply --yes
`),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cfg, err := analysisContextAndConfig(cmd, config, skipCacheFlag)
			if err != nil {
				return err
			}
			configured := configuredProjectIDs(cfg)
			unionMode := strings.EqualFold(sourceProjectID, "all")
			if sourceProjectID == "" {
				return fmt.Errorf("--source is required")
			}
			if len(targetProjectIDs) == 0 {
				if unionMode {
					targetProjectIDs = configured
				} else {
					targetProjectIDs = projectsExcept(configured, sourceProjectID)
				}
			}

			planOpts := core.FlagSyncOptions{
				SourceProjectID:  sourceProjectID,
				TargetProjectIDs: uniqueNonEmpty(targetProjectIDs),
				FlagKeys:         uniqueNonEmpty(flagKeys),
				UpdateVariables:  updateVars,
				UnionSource:      unionMode,
				SyncVariations:   syncVariations,
			}

			var plan *core.FlagSyncPlan
			err = loader.Run("Planning flag sync...", func() {
				plan, err = s.PlanFlagSync(ctx, planOpts)
			})
			if err != nil {
				return err
			}

			if !apply {
				if jsonOutput {
					return writeJSONOutput(plan, toFilePath, stdout)
				}
				return writeAnalysisOutput(tui.RenderOptimizelySyncPlan(*plan, true, nil), toFilePath, stdout)
			}

			changeCount := syncChangeCount(*plan)
			if changeCount == 0 {
				if jsonOutput {
					return writeJSONOutput(struct {
						Plan   *core.FlagSyncPlan   `json:"plan"`
						Result *core.FlagSyncResult `json:"result,omitempty"`
					}{Plan: plan}, toFilePath, stdout)
				}
				return writeAnalysisOutput(tui.RenderOptimizelySyncPlan(*plan, false, nil), toFilePath, stdout)
			}

			if !yes {
				confirmed, err := confirmSyncApply(changeCount)
				if err != nil {
					return err
				}
				if !confirmed {
					writer.Info("Aborted.")
					return nil
				}
			}

			var result *core.FlagSyncResult
			err = loader.Run("Applying flag sync...", func() {
				result, err = s.ApplyFlagSyncPlan(ctx, *plan)
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				return writeJSONOutput(struct {
					Plan   *core.FlagSyncPlan   `json:"plan"`
					Result *core.FlagSyncResult `json:"result,omitempty"`
				}{Plan: plan, Result: result}, toFilePath, stdout)
			}
			return writeAnalysisOutput(tui.RenderOptimizelySyncPlan(*plan, false, result), toFilePath, stdout)
		},
	}

	cmd.Flags().StringVar(&sourceProjectID, "source", "", "Source project ID, or 'all' for union mode")
	cmd.Flags().StringSliceVar(&targetProjectIDs, "target", nil, "Target project IDs (defaults to other configured projects, or all configured for union mode)")
	cmd.Flags().StringSliceVar(&flagKeys, "flag", nil, "Restrict sync to specific flag keys")
	cmd.Flags().BoolVar(&updateVars, "update-vars", false, "Also add missing variable definitions to flags that already exist")
	cmd.Flags().BoolVar(&apply, "apply", false, "Apply the sync plan. Without this flag, a dry-run plan is shown")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip the live apply confirmation prompt")
	cmd.Flags().BoolVar(&skipCacheFlag, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Dump output directly to stdout without pager")
	cmd.Flags().StringVar(&toFilePath, "to-file", "", "Write output to the specified file path")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write JSON output")
	cmd.Flags().BoolVar(&syncVariations, "sync-variations", false, "Also duplicate all flag variations/overrides when creating missing flags")
	registerSourceProjectCompletion(cmd, config)
	registerProjectFlagCompletion(cmd, config, "target")
	registerFlagKeyCompletion(cmd, s, config, "source", "flag")
	_ = cmd.MarkFlagRequired("source")
	return cmd
}

func analysisContextAndConfig(cmd *cobra.Command, config *Config, skipCacheFlag bool) (context.Context, *Configuration, error) {
	ctx := cmd.Context()
	if skipCacheFlag {
		ctx = context.WithValue(ctx, driven.BypassCacheKey, true)
	}
	cfg, err := config.Get()
	if err != nil {
		return nil, nil, err
	}
	if cfg.APIKey == "" {
		return nil, nil, fmt.Errorf("API key is required")
	}
	return ctx, cfg, nil
}

func configuredProjectIDs(cfg *Configuration) []string {
	ids := make([]string, 0, len(cfg.Projects))
	for _, project := range cfg.Projects {
		ids = append(ids, project.ID)
	}
	return uniqueNonEmpty(ids)
}

func mergeProjectSelections(configured []string, explicit []string, args []string) []string {
	combined := make([]string, 0, len(explicit)+len(args))
	combined = append(combined, explicit...)
	combined = append(combined, args...)
	combined = uniqueNonEmpty(combined)
	if len(combined) > 0 {
		return combined
	}
	return uniqueNonEmpty(configured)
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func projectsExcept(projectIDs []string, excluded string) []string {
	result := make([]string, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		if projectID != excluded {
			result = append(result, projectID)
		}
	}
	return result
}

func writeAnalysisOutput(content, toFilePath string, stdout bool) error {
	if toFilePath != "" {
		return os.WriteFile(toFilePath, []byte(content), 0644)
	}
	if stdout || !loader.IsTerminal() {
		fmt.Print(content)
		return nil
	}
	return tui.RunPager(content)
}

func writeJSONOutput(value any, toFilePath string, stdout bool) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if toFilePath != "" {
		return os.WriteFile(toFilePath, data, 0644)
	}
	if stdout || !loader.IsTerminal() {
		fmt.Println(string(data))
		return nil
	}
	return tui.RunPager(string(data))
}

func confirmSyncApply(changeCount int) (bool, error) {
	if !loader.IsTerminal() {
		return false, fmt.Errorf("--yes is required when applying sync in a non-interactive environment")
	}
	confirmed := false
	err := tui.NewForm(
		huh.NewGroup(
			huh.NewSelect[bool]().
				Title(fmt.Sprintf("Apply %d Optimizely sync change(s)?", changeCount)).
				Options(
					huh.NewOption("Yes", true),
					huh.NewOption("No", false),
				).
				Value(&confirmed),
		),
	).Run()
	return confirmed, err
}

func syncChangeCount(plan core.FlagSyncPlan) int {
	count := 0
	for _, target := range plan.TargetMissing {
		count += len(target.Flags)
	}
	for _, target := range plan.TargetVariableUpdates {
		for _, update := range target.Updates {
			count += len(update.MissingVariables)
		}
	}
	return count
}
