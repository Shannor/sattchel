package driving

import (
	"context"
	"fmt"
	"os"
	"sattchel/internal/optimizely/adapters/driven"
	"sattchel/internal/optimizely/core"
	"sattchel/internal/printer"
	"sattchel/internal/tui"
	"slices"
	"strings"

	"sattchel/pkg/loader"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var (
	projectFilter    = make([]string, 0)
	envFilter        = make([]string, 0)
	queryFilter      string
	skipCache        bool
	showDetails      bool
	showVariants     bool
	showEnvironments bool
	showVariables    bool
	stdoutFlag       bool
	toFile           string
)

func flags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var flagCmd = &cobra.Command{
		Use:          "flags",
		Short:        "Manage feature flags",
		Aliases:      []string{"ff"},
		SilenceUsage: true,
	}

	flagCmd.AddCommand(listFlags(s, config, writer))
	flagCmd.AddCommand(getFlag(s, config))
	flagCmd.AddCommand(compareFlags(s, config, writer))
	flagCmd.AddCommand(uniqueFlags(s, config, writer))
	flagCmd.AddCommand(dormantFlags(s, config, writer))
	flagCmd.AddCommand(driftFlags(s, config, writer))
	flagCmd.AddCommand(syncFlags(s, config, writer))
	return flagCmd
}

func getFlag(s *core.Service, config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get feature flag details",
		Args:  cobra.MaximumNArgs(1),
		Long: `Get all details about a feature flag, including variations, statuses,
variables, and usage across one or more configured projects.`,
		Example: strings.TrimSpace(`
  satt optimizely flags get checkout_redesign
  satt optimizely flags get checkout_redesign --project 123
  satt optimizely flags get checkout_redesign --project 123 --stdout
`),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("must provide a feature flag key/id")
			}
			flagId := args[0]
			ctx := cmd.Context()
			if skipCache {
				ctx = context.WithValue(ctx, driven.BypassCacheKey, true)
			}
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				return fmt.Errorf("API key is required")
			}
			projects := cfg.Projects
			if len(projects) == 0 {
				return fmt.Errorf("no projects configured")
			}

			var targetProjectIDs []string
			if len(projectFilter) > 0 {
				targetProjectIDs = projectFilter
			} else {
				for _, project := range projects {
					targetProjectIDs = append(targetProjectIDs, project.ID)
				}
			}

			var reports []tui.ProjectFlagReport
			var lastErr error

			projMap := make(map[string]core.Project)
			for _, p := range projects {
				projMap[p.ID] = p
			}

			for _, pid := range targetProjectIDs {
				environments := []string{"production", "demo", "preprod", "qa", "development"}
				if envs, ok := cfg.EnvironmentMap[pid]; ok && len(envs) > 0 {
					environments = make([]string, 0, len(envs))
					for _, env := range envs {
						if !env.Archived {
							environments = append(environments, env.Key)
						}
					}
				}

				flag, instances, err := s.GetFlag(ctx, pid, environments, flagId)
				if err != nil {
					lastErr = err
					continue
				}

				proj, ok := projMap[pid]
				if !ok {
					proj = core.Project{ID: pid, Name: pid}
				}

				reports = append(reports, tui.ProjectFlagReport{
					Project:   proj,
					Flag:      flag,
					Instances: instances,
				})
			}

			if len(reports) == 0 {
				if lastErr != nil {
					return fmt.Errorf("feature flag %q not found or failed to fetch: %w", flagId, lastErr)
				}
				return fmt.Errorf("feature flag %q not found in any of the checked projects", flagId)
			}

			opts := tui.ReportOptions{
				ShowDetails:      showDetails,
				ShowVariants:     showVariants,
				ShowEnvironments: showEnvironments,
				ShowVariables:    showVariables,
			}

			content, renderErr := tui.RenderMultiProjectFlagLipGlossStr(reports, opts)
			if renderErr != nil {
				return renderErr
			}

			bypassPager := stdoutFlag
			if toFile != "" {
				err := os.WriteFile(toFile, []byte(content), 0644)
				if err != nil {
					return fmt.Errorf("failed to write to file %s: %w", toFile, err)
				}
				return nil
			}

			if bypassPager {
				fmt.Print(content)
				return nil
			}

			return tui.RunPager(content)
		},
	}
	cmd.Flags().StringArrayVar(&envFilter, "env", []string{}, "if provided will only show the flag for the environment(s) (if not provided will show all)")
	cmd.Flags().StringArrayVarP(&projectFilter, "project", "p", []string{}, "if provided will only show the flag for the project(s) (if not provided will show all)")
	registerProjectFlagCompletion(cmd, config, "project")
	registerGetFlagArgCompletion(cmd, s, config, "project")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&showDetails, "show-details", true, "Show flag details (ID, status, etc.)")
	cmd.Flags().BoolVar(&showVariants, "show-variants", true, "Show variation/variant definitions")
	cmd.Flags().BoolVar(&showEnvironments, "show-environments", true, "Show environment configurations")
	cmd.Flags().BoolVar(&showVariables, "show-variables", true, "Show variable definitions and overrides")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump output directly to stdout without pager")
	cmd.Flags().StringVar(&toFile, "to-file", "", "Write output to the specified file path")
	return cmd
}

func listFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List feature flags between projects",
		Long:  "List feature flags across the selected projects, optionally filtered by query.",
		Example: strings.TrimSpace(`
  satt optimizely flags list
  satt optimizely flags list --project 123 --project 456 --query loyalty
  satt optimizely flags list --stdout
`),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if skipCache {
				ctx = context.WithValue(ctx, driven.BypassCacheKey, true)
			}
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				return fmt.Errorf("API key is required")
			}
			projects := cfg.Projects
			ids := make([]string, 0)
			if len(projectFilter) > 0 {
				ids = projectFilter
			} else {
				for _, project := range projects {
					ids = append(ids, project.ID)
				}
			}

			var flags map[string][]core.FeatureFlagDefinition
			if stdoutFlag || toFile != "" {
				if queryFilter != "" {
					flags, err = s.SearchFlags(ctx, ids, core.ListFlagsOptions{Query: queryFilter})
				} else {
					flags, err = s.GetFlags(ctx, ids)
				}
			} else {
				err = loader.Run("Retrieving feature flags...", func() {
					if queryFilter != "" {
						flags, err = s.SearchFlags(ctx, ids, core.ListFlagsOptions{Query: queryFilter})
					} else {
						flags, err = s.GetFlags(ctx, ids)
					}
				})
			}

			if err != nil {
				return err
			}

			// Map project IDs to project Names
			projMap := make(map[string]string)
			for _, p := range cfg.Projects {
				projMap[p.ID] = p.Name
			}

			type flagGroup struct {
				Key      string
				Name     string
				Projects []string
			}
			groups := make(map[string]*flagGroup)

			for pid, flagList := range flags {
				pName := projMap[pid]
				if pName == "" {
					pName = pid
				}
				for _, f := range flagList {
					if g, ok := groups[f.Key]; ok {
						found := false
						for _, p := range g.Projects {
							if p == pName {
								found = true
								break
							}
						}
						if !found {
							g.Projects = append(g.Projects, pName)
						}
					} else {
						groups[f.Key] = &flagGroup{
							Key:      f.Key,
							Name:     f.Name,
							Projects: []string{pName},
						}
					}
				}
			}

			// Sort project lists and keys
			var sortedKeys []string
			for k, g := range groups {
				slices.Sort(g.Projects)
				sortedKeys = append(sortedKeys, k)
			}
			slices.SortFunc(sortedKeys, func(a, b string) int {
				return strings.Compare(strings.ToLower(a), strings.ToLower(b))
			})

			var options []tui.ListOption
			for _, k := range sortedKeys {
				g := groups[k]
				options = append(options, tui.ListOption{
					TitleStr:       g.Key,
					DescriptionStr: strings.Join(g.Projects, ", "),
					ValueStr:       g.Key,
				})
			}

			if len(options) == 0 {
				writer.Info("No feature flags found.")
				return nil
			}

			bypassUI := stdoutFlag || toFile != "" || !loader.IsTerminal()

			if bypassUI {
				var sb strings.Builder
				for _, k := range sortedKeys {
					g := groups[k]
					sb.WriteString(fmt.Sprintf("%s: %s\n", g.Key, strings.Join(g.Projects, ", ")))
				}
				content := sb.String()

				if toFile != "" {
					err := os.WriteFile(toFile, []byte(content), 0644)
					if err != nil {
						return fmt.Errorf("failed to write to file %s: %w", toFile, err)
					}
					return nil
				}

				fmt.Print(content)
				return nil
			}

			selected, err := tui.Choose("Select Feature Flag to copy 'get' command", options)
			if err != nil {
				return err
			}

			var selectedKey string
			if selected != nil {
				selectedKey = selected.ValueStr
			}

			if selectedKey != "" {
				cmdStr := fmt.Sprintf("satt optimizely flags get %s", selectedKey)
				if err := clipboard.WriteAll(cmdStr); err != nil {
					writer.Error(fmt.Sprintf("Selected: %s\n(Failed to copy to clipboard: %v)\n", selectedKey, err))
				} else {
					writer.Success(fmt.Sprintf("Selected: %s\nCopied to clipboard: %s\n", selectedKey, cmdStr))
				}
			}

			return nil
		},
	}
	cmd.Flags().StringArrayVarP(&projectFilter, "project", "p", []string{}, "if provided will only show the flags for the provided project ids. (if not provided will show all)")
	registerProjectFlagCompletion(cmd, config, "project")
	cmd.Flags().StringArrayVar(&envFilter, "env", []string{}, "if provided will only show the flag for the environment(s) (if not provided will show all)")
	cmd.Flags().StringVar(&queryFilter, "query", "", "Filter the flags by name, key, or description substring")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump list directly to stdout instead of interactive UI")
	cmd.Flags().StringVar(&toFile, "to-file", "", "Write list to the specified file path")
	return cmd
}

func compareFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var (
		baseProjectID  string
		focusProjectID string
		jsonOutput     bool
	)

	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare feature flags across multiple projects and list missing flags",
		Long: `Compare feature flags across 2 or more projects.
Finds and returns a list of feature flags that don't exist in all of the specified project IDs.
If no project IDs are provided, project IDs saved in the configuration will be used.
There must be at least 2 project IDs provided or saved in the configuration.`,
		Args: cobra.NoArgs,
		Example: strings.TrimSpace(`
  satt optimizely flags compare --project 123 --project 456
  satt optimizely flags compare --project 123 --project 456 --query loyalty
  satt optimizely flags compare --base 123 --project 456 --project 789
  satt optimizely flags compare --focus 456 --project 123 --project 456 --project 789 --json
`),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if skipCache {
				ctx = context.WithValue(ctx, driven.BypassCacheKey, true)
			}
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			if cfg.APIKey == "" {
				return fmt.Errorf("API key is required")
			}

			targetProjectIDs := projectFilter
			if len(targetProjectIDs) == 0 {
				targetProjectIDs = configuredProjectIDs(cfg)
			}
			if baseProjectID != "" && !slices.Contains(targetProjectIDs, baseProjectID) {
				targetProjectIDs = append([]string{baseProjectID}, targetProjectIDs...)
			}
			if len(targetProjectIDs) < 2 {
				return fmt.Errorf("at least 2 project IDs are required for comparison (found: %d)", len(targetProjectIDs))
			}

			var report *core.FlagComparisonReport
			err = loader.Run("Comparing feature flags...", func() {
				report, err = s.CompareFlagsDetailed(ctx, targetProjectIDs, core.CompareFlagsOptions{
					Query:          queryFilter,
					BaseProjectID:  baseProjectID,
					FocusProjectID: focusProjectID,
				})
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				return writeJSONOutput(report, toFile, stdoutFlag)
			}

			content := tui.RenderOptimizelyCompareReport(report, baseProjectID, focusProjectID)
			if len(report.MissingFlags) == 0 {
				writer.Success("All feature flags match perfectly across all checked projects!")
			}
			return writeAnalysisOutput(content, toFile, stdoutFlag)
		},
	}
	cmd.Flags().StringArrayVarP(&projectFilter, "project", "p", []string{}, "if provided, compares only the specified project(s)")
	registerProjectFlagCompletion(cmd, config, "project", "base", "focus")
	cmd.Flags().StringVar(&baseProjectID, "base", "", "Show only flags missing from this base project")
	cmd.Flags().StringVar(&focusProjectID, "focus", "", "Show only flags this focus project has that at least one other project lacks")
	cmd.Flags().StringVar(&queryFilter, "query", "", "Filter the flags by name, key, or description substring")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write JSON output")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump list directly to stdout instead of using a pager")
	cmd.Flags().StringVar(&toFile, "to-file", "", "Write comparison list to the specified file path")
	return cmd
}
