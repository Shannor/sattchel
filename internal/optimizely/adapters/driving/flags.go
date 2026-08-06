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
	"slices"
	"strconv"
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
	outputFormat     string
	showDetails      bool
	showVariants     bool
	showEnvironments bool
	stdoutFlag       bool
	toFile           string
)

func cmdFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var flagCmd = &cobra.Command{
		Use:          "flags",
		Short:        "Manage feature flags",
		Aliases:      []string{"ff"},
		SilenceUsage: true,
	}

	flagCmd.AddCommand(listFlags(s, config, writer))
	flagCmd.AddCommand(getFlag(s, config))
	flagCmd.AddCommand(compareFlags(s, config, writer))
	flagCmd.AddCommand(syncFlag(s, config, writer))
	return flagCmd
}

func getFlag(s *core.Service, config *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get feature flag details",
		Args:  cobra.MaximumNArgs(1),
		Long: `Get's all details about a feature flag.
 	Including information like the variations, statuses, and usage per project
   Examples:
     satt optimizely flags get <key>
     `,
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
	cmd.Flags().StringArrayVar(&projectFilter, "project", []string{}, "if provided will only show the flag for the project(s) (if not provided will show all)")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&showDetails, "show-details", true, "Show flag details (ID, status, etc.)")
	cmd.Flags().BoolVar(&showVariants, "show-variants", true, "Show variation/variant definitions")
	cmd.Flags().BoolVar(&showEnvironments, "show-environments", true, "Show environment configurations")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump output directly to stdout without pager")
	cmd.Flags().StringVar(&toFile, "to-file", "", "Write output to the specified file path")
	return cmd
}

func listFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List feature flags between projects",
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
	cmd.Flags().StringArrayVar(&projectFilter, "filter", []string{}, "if provided will only show the flags for the provided project ids. (if not provided will show all)")
	cmd.Flags().StringArrayVar(&envFilter, "env", []string{}, "if provided will only show the flag for the environment(s) (if not provided will show all)")
	cmd.Flags().StringVar(&queryFilter, "query", "", "Filter the flags by name, key, or description substring")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump list directly to stdout instead of interactive UI")
	cmd.Flags().StringVar(&toFile, "to-file", "", "Write list to the specified file path")
	return cmd
}

func compareFlags(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare [project-ids...]",
		Short: "Compare feature flags across multiple projects and list missing flags",
		Long: `Compare feature flags across 2 or more projects.
Finds and returns a list of feature flags that don't exist in all of the specified project IDs.
If no project IDs are provided as arguments, project IDs saved in the configuration will be used.
There must be at least 2 project IDs provided or saved in the configuration.`,
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

			// Gather project IDs from flags/args
			var targetProjectIDs []string
			if len(projectFilter) > 0 {
				targetProjectIDs = projectFilter
			}
			if len(args) > 0 {
				targetProjectIDs = append(targetProjectIDs, args...)
			}
			if len(targetProjectIDs) == 0 {
				for _, project := range cfg.Projects {
					targetProjectIDs = append(targetProjectIDs, project.ID)
				}
			}

			// Check minimum requirement
			if len(targetProjectIDs) < 2 {
				return fmt.Errorf("at least 2 project IDs are required for comparison (found: %d)", len(targetProjectIDs))
			}

			var comparisons []core.FlagComparison
			if stdoutFlag || toFile != "" {
				comparisons, err = s.CompareFlags(ctx, targetProjectIDs)
			} else {
				err = loader.Run("Comparing feature flags...", func() {
					comparisons, err = s.CompareFlags(ctx, targetProjectIDs)
				})
			}

			if err != nil {
				return err
			}

			if len(comparisons) == 0 {
				writer.Success("All feature flags match perfectly across all checked projects!")
				return nil
			}

			var content string
			var renderErr error
			if outputFormat == "lipgloss" {
				content, renderErr = tui.RenderFlagComparisonsLipGlossStr(comparisons)
			} else {
				// default to markdown/glamour
				content, renderErr = tui.RenderFlagComparisonsGlamourStr(comparisons)
			}
			if renderErr != nil {
				return renderErr
			}

			if toFile != "" {
				err := os.WriteFile(toFile, []byte(content), 0644)
				if err != nil {
					return fmt.Errorf("failed to write to file %s: %w", toFile, err)
				}
				return nil
			}

			if stdoutFlag || !loader.IsTerminal() {
				fmt.Print(content)
				return nil
			}

			return tui.RunPager(content)
		},
	}
	cmd.Flags().StringArrayVar(&projectFilter, "project", []string{}, "if provided, compares only the specified project(s)")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "markdown", "Output format (markdown, lipgloss)")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump list directly to stdout instead of using a pager")
	cmd.Flags().StringVar(&toFile, "to-file", "", "Write comparison list to the specified file path")
	return cmd
}

type mismatchInfo struct {
	Project     core.Project
	Differences []string
}

func syncFlag(s *core.Service, config *Config, writer printer.Writer) *cobra.Command {
	var apply bool
	var sourceProjID string

	cmd := &cobra.Command{
		Use:   "sync <key>",
		Short: "Sync a feature flag across projects",
		Long: `Sync a feature flag across multiple projects.
It fetches the full flag definition (including variations and variables) from a source project where the flag exists,
and creates it in target projects where the flag is currently missing.
By default, it runs in dry-run mode, showing a plan. Pass --apply to perform the sync.`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flagKey := args[0]
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

			// Gather all projects
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

			if len(targetProjectIDs) < 2 {
				return fmt.Errorf("at least 2 projects are required for syncing (found: %d)", len(targetProjectIDs))
			}

			// Use s.CompareFlags to determine where the flag exists and where it is missing
			var comparisons []core.FlagComparison
			if stdoutFlag {
				comparisons, err = s.CompareFlags(ctx, targetProjectIDs)
			} else {
				err = loader.Run("Comparing feature flags...", func() {
					comparisons, err = s.CompareFlags(ctx, targetProjectIDs)
				})
			}
			if err != nil {
				return err
			}

			// Find the comparison for our target flag key
			var targetComp *core.FlagComparison
			for i := range comparisons {
				if comparisons[i].Key == flagKey {
					targetComp = &comparisons[i]
					break
				}
			}

			existsIn := make([]core.Project, 0)
			missingIn := make([]core.Project, 0)

			projMap := make(map[string]core.Project)
			for _, p := range projects {
				projMap[p.ID] = p
			}

			if targetComp != nil {
				existsIn = targetComp.ExistsIn
				missingIn = targetComp.MissingIn
			} else {
				// The flag was not in comparisons, meaning it either exists in all or none.
				// Let's verify by trying to fetch the flag from target projects.
				for _, pid := range targetProjectIDs {
					_, _, getErr := s.GetFlag(ctx, pid, nil, flagKey)
					p := projMap[pid]
					if getErr == nil {
						existsIn = append(existsIn, p)
					} else {
						missingIn = append(missingIn, p)
					}
				}
			}

			if len(existsIn) == 0 {
				return fmt.Errorf("feature flag %q does not exist in any of the checked projects; cannot sync", flagKey)
			}

			// Determine source project
			var sourceProj core.Project
			var otherSources []string
			if sourceProjID != "" {
				found := false
				for _, p := range existsIn {
					if p.ID == sourceProjID {
						sourceProj = p
						found = true
					} else {
						otherSources = append(otherSources, fmt.Sprintf("%s (%s)", p.Name, p.ID))
					}
				}
				if !found {
					return fmt.Errorf("source project %s does not contain feature flag %q", sourceProjID, flagKey)
				}
			} else {
				// Default to first project where it exists
				sourceProj = existsIn[0]
				for _, p := range existsIn[1:] {
					otherSources = append(otherSources, fmt.Sprintf("%s (%s)", p.Name, p.ID))
				}
			}

			// Fetch the fully enriched flag from the source project
			var sourceFlag *core.FeatureFlagDefinition
			if stdoutFlag {
				sourceFlag, _, err = s.GetFlag(ctx, sourceProj.ID, nil, flagKey)
			} else {
				err = loader.Run(fmt.Sprintf("Fetching enriched flag from %s...", sourceProj.Name), func() {
					sourceFlag, _, err = s.GetFlag(ctx, sourceProj.ID, nil, flagKey)
				})
			}
			if err != nil {
				return fmt.Errorf("failed to fetch enriched flag from source project %s: %w", sourceProj.Name, err)
			}

			// Check configuration differences for projects where it already exists
			var mismatches []mismatchInfo

			for _, p := range existsIn {
				if p.ID == sourceProj.ID {
					continue
				}
				var existingFlag *core.FeatureFlagDefinition
				if stdoutFlag {
					existingFlag, _, err = s.GetFlag(ctx, p.ID, nil, flagKey)
				} else {
					err = loader.Run(fmt.Sprintf("Fetching flag config from %s for comparison...", p.Name), func() {
						existingFlag, _, err = s.GetFlag(ctx, p.ID, nil, flagKey)
					})
				}
				if err != nil {
					writer.Info(fmt.Sprintf("Warning: failed to fetch enriched flag from %s for comparison: %v", p.Name, err))
					continue
				}

				diffs := findConfigurationDifferences(sourceFlag, existingFlag)
				if len(diffs) > 0 {
					mismatches = append(mismatches, mismatchInfo{
						Project:     p,
						Differences: diffs,
					})
				}
			}

			// Display Sync Plan
			planContent := renderSyncPlan(sourceFlag, sourceProj, missingIn, otherSources, mismatches)
			fmt.Print(planContent)

			if len(missingIn) == 0 {
				if len(mismatches) > 0 {
					writer.Info("All target projects already contain the feature flag, but configuration differences exist.")
				} else {
					writer.Success("Feature flag configurations are perfectly aligned across all projects. Nothing to sync!")
				}
				return nil
			}

			if !apply {
				writer.Info("[Dry Run] Sync plan generated. No changes were made.")
				writer.Info("To apply these changes and create the flag in missing projects, run the command with the --apply flag.")
				return nil
			}

			// Perform the creation
			for _, p := range missingIn {
				writer.Info(fmt.Sprintf("Creating feature flag %q in project %s (ID: %s)...", flagKey, p.Name, p.ID))
				if stdoutFlag {
					_, err = s.CreateFlag(ctx, p.ID, *sourceFlag)
				} else {
					err = loader.Run(fmt.Sprintf("Syncing to %s...", p.Name), func() {
						_, err = s.CreateFlag(ctx, p.ID, *sourceFlag)
					})
				}
				if err != nil {
					writer.Error(fmt.Sprintf("Failed to sync to project %s: %v", p.Name, err))
					return err
				}
				writer.Success(fmt.Sprintf("Successfully synced to project %s!", p.Name))
			}

			writer.Success("Feature flag sync complete!")
			return nil
		},
	}

	cmd.Flags().StringVar(&sourceProjID, "source", "", "The project ID to copy the flag from")
	cmd.Flags().StringArrayVar(&projectFilter, "project", []string{}, "Target project IDs to sync across (defaults to all configured projects)")
	cmd.Flags().BoolVar(&apply, "apply", false, "Actually apply the sync changes (default is dry-run)")
	cmd.Flags().BoolVar(&skipCache, "skip-cache", false, "Skip the feature flag cache and fetch fresh data from Optimizely")
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Dump output directly to stdout without loaders/spinners")
	return cmd
}

func findConfigurationDifferences(source *core.FeatureFlagDefinition, target *core.FeatureFlagDefinition) []string {
	var diffs []string

	// Compare variables
	checkVar := func(key string, sourceVal, targetVal string, sourceType, targetType string) {
		if sourceType != targetType {
			diffs = append(diffs, fmt.Sprintf("variable %q: type mismatch (source: %s, target: %s)", key, sourceType, targetType))
		} else if sourceVal != targetVal {
			diffs = append(diffs, fmt.Sprintf("variable %q: default value mismatch (source: %s, target: %s)", key, sourceVal, targetVal))
		}
	}

	sourceKeys := make(map[string]bool)

	for k, v := range source.DefaultVariables.BoolVariables {
		sourceKeys[k] = true
		if tv, ok := target.DefaultVariables.BoolVariables[k]; ok {
			checkVar(k, strconv.FormatBool(v.Value), strconv.FormatBool(tv.Value), v.Type, tv.Type)
		} else {
			diffs = append(diffs, fmt.Sprintf("variable %q: missing in target project", k))
		}
	}
	for k, v := range source.DefaultVariables.IntVariables {
		sourceKeys[k] = true
		if tv, ok := target.DefaultVariables.IntVariables[k]; ok {
			checkVar(k, strconv.Itoa(v.Value), strconv.Itoa(tv.Value), v.Type, tv.Type)
		} else {
			diffs = append(diffs, fmt.Sprintf("variable %q: missing in target project", k))
		}
	}
	for k, v := range source.DefaultVariables.FloatVariables {
		sourceKeys[k] = true
		if tv, ok := target.DefaultVariables.FloatVariables[k]; ok {
			checkVar(k, strconv.FormatFloat(v.Value, 'f', -1, 64), strconv.FormatFloat(tv.Value, 'f', -1, 64), v.Type, tv.Type)
		} else {
			diffs = append(diffs, fmt.Sprintf("variable %q: missing in target project", k))
		}
	}
	for k, v := range source.DefaultVariables.StringVariables {
		sourceKeys[k] = true
		if tv, ok := target.DefaultVariables.StringVariables[k]; ok {
			checkVar(k, v.Value, tv.Value, v.Type, tv.Type)
		} else {
			diffs = append(diffs, fmt.Sprintf("variable %q: missing in target project", k))
		}
	}
	for k, v := range source.DefaultVariables.JsonVariables {
		sourceKeys[k] = true
		if tv, ok := target.DefaultVariables.JsonVariables[k]; ok {
			sourceBytes, _ := json.Marshal(v.Value)
			targetBytes, _ := json.Marshal(tv.Value)
			checkVar(k, string(sourceBytes), string(targetBytes), v.Type, tv.Type)
		} else {
			diffs = append(diffs, fmt.Sprintf("variable %q: missing in target project", k))
		}
	}

	// Check target extra variables
	for k := range target.DefaultVariables.BoolVariables {
		if !sourceKeys[k] {
			diffs = append(diffs, fmt.Sprintf("variable %q: exists in target but not in source project", k))
		}
	}
	for k := range target.DefaultVariables.IntVariables {
		if !sourceKeys[k] {
			diffs = append(diffs, fmt.Sprintf("variable %q: exists in target but not in source project", k))
		}
	}
	for k := range target.DefaultVariables.FloatVariables {
		if !sourceKeys[k] {
			diffs = append(diffs, fmt.Sprintf("variable %q: exists in target but not in source project", k))
		}
	}
	for k := range target.DefaultVariables.StringVariables {
		if !sourceKeys[k] {
			diffs = append(diffs, fmt.Sprintf("variable %q: exists in target but not in source project", k))
		}
	}
	for k := range target.DefaultVariables.JsonVariables {
		if !sourceKeys[k] {
			diffs = append(diffs, fmt.Sprintf("variable %q: exists in target but not in source project", k))
		}
	}

	// Compare overrides
	sourceOverrides := make(map[string]core.Override)
	for _, o := range source.Overrides {
		sourceOverrides[o.Key] = o
	}
	targetOverrides := make(map[string]core.Override)
	for _, o := range target.Overrides {
		targetOverrides[o.Key] = o
	}

	for k, so := range sourceOverrides {
		if to, ok := targetOverrides[k]; ok {
			if so.Name != to.Name {
				diffs = append(diffs, fmt.Sprintf("variation %q: name mismatch (source: %s, target: %s)", k, so.Name, to.Name))
			}
			soBytes, _ := so.Variables.MarshalJSON()
			toBytes, _ := to.Variables.MarshalJSON()
			if string(soBytes) != string(toBytes) {
				diffs = append(diffs, fmt.Sprintf("variation %q: variable values mismatch", k))
			}
		} else {
			diffs = append(diffs, fmt.Sprintf("variation %q: missing in target project", k))
		}
	}

	for k := range targetOverrides {
		if _, ok := sourceOverrides[k]; !ok {
			diffs = append(diffs, fmt.Sprintf("variation %q: exists in target but not in source project", k))
		}
	}

	return diffs
}

func renderSyncPlan(sourceFlag *core.FeatureFlagDefinition, sourceProj core.Project, missingIn []core.Project, otherSources []string, mismatches []mismatchInfo) string {
	var sb strings.Builder
	s := tui.AutoStyles()

	sb.WriteString("\n")
	sb.WriteString(s.Title.Render(" ⚡ FEATURE FLAG SYNC PLAN ") + "\n")
	sb.WriteString("\n")

	sb.WriteString(s.Info.Bold(true).Render("  Flag Information:") + "\n")
	sb.WriteString(fmt.Sprintf("    • %s: %s\n", s.Muted.Render("Key"), s.Text.Bold(true).Render(sourceFlag.Key)))
	sb.WriteString(fmt.Sprintf("    • %s: %s\n", s.Muted.Render("Name"), s.Text.Render(sourceFlag.Name)))
	if sourceFlag.Description != "" {
		sb.WriteString(fmt.Sprintf("    • %s: %s\n", s.Muted.Render("Description"), s.Text.Render(sourceFlag.Description)))
	}

	var sourceRationale string
	if len(otherSources) > 0 {
		sourceRationale = fmt.Sprintf("%s (selected automatically; also exists in: %s)", sourceProj.Name, strings.Join(otherSources, ", "))
	} else {
		sourceRationale = fmt.Sprintf("%s (only project containing this flag)", sourceProj.Name)
	}
	sb.WriteString(fmt.Sprintf("    • %s: %s\n", s.Muted.Render("Source Project"), s.Text.Render(sourceRationale)))
	sb.WriteString("\n")

	sb.WriteString(s.Info.Bold(true).Render("  Variables to Create/Sync:") + "\n")
	hasVars := false
	renderVar := func(key, typ string, val any, desc string) {
		hasVars = true
		sb.WriteString(fmt.Sprintf("    • %s (%s):\n", s.Text.Bold(true).Render(key), s.Muted.Render(typ)))
		if desc != "" {
			sb.WriteString(fmt.Sprintf("        Description: %s\n", desc))
		}
		if typ == "json" {
			sb.WriteString("        Value:\n")
			formatted := tui.MarshalJSON(val)
			for _, line := range strings.Split(formatted, "\n") {
				sb.WriteString(fmt.Sprintf("          %s\n", line))
			}
		} else {
			sb.WriteString(fmt.Sprintf("        Value: %v\n", val))
		}
	}

	for k, v := range sourceFlag.DefaultVariables.BoolVariables {
		renderVar(k, "boolean", v.Value, v.Description)
	}
	for k, v := range sourceFlag.DefaultVariables.IntVariables {
		renderVar(k, "integer", v.Value, v.Description)
	}
	for k, v := range sourceFlag.DefaultVariables.FloatVariables {
		renderVar(k, "float", v.Value, v.Description)
	}
	for k, v := range sourceFlag.DefaultVariables.StringVariables {
		renderVar(k, "string", fmt.Sprintf("%q", v.Value), v.Description)
	}
	for k, v := range sourceFlag.DefaultVariables.JsonVariables {
		renderVar(k, "json", v.Value, v.Description)
	}
	if !hasVars {
		sb.WriteString("    (None)\n")
	}
	sb.WriteString("\n")

	sb.WriteString(s.Info.Bold(true).Render("  Variations to Create/Sync:") + "\n")
	if len(sourceFlag.Overrides) == 0 {
		sb.WriteString("    (None)\n")
	} else {
		for _, override := range sourceFlag.Overrides {
			sb.WriteString(fmt.Sprintf("    • %s (%s):\n", s.Text.Bold(true).Render(override.Key), s.Muted.Render(override.Name)))
			if override.Description != "" {
				sb.WriteString(fmt.Sprintf("        Description: %s\n", override.Description))
			}

			hasOverrideVars := false
			renderOverrideVar := func(key, typ string, val any) {
				if !hasOverrideVars {
					sb.WriteString("        Variables:\n")
					hasOverrideVars = true
				}
				sb.WriteString(fmt.Sprintf("          * %s (%s):\n", s.Text.Bold(true).Render(key), s.Muted.Render(typ)))
				if typ == "json" {
					formatted := tui.MarshalJSON(val)
					for _, line := range strings.Split(formatted, "\n") {
						sb.WriteString(fmt.Sprintf("              %s\n", line))
					}
				} else {
					sb.WriteString(fmt.Sprintf("              Value: %v\n", val))
				}
			}

			for k, v := range override.Variables.BoolVariables {
				renderOverrideVar(k, "boolean", v.Value)
			}
			for k, v := range override.Variables.IntVariables {
				renderOverrideVar(k, "integer", v.Value)
			}
			for k, v := range override.Variables.FloatVariables {
				renderOverrideVar(k, "float", v.Value)
			}
			for k, v := range override.Variables.StringVariables {
				renderOverrideVar(k, "string", fmt.Sprintf("%q", v.Value))
			}
			for k, v := range override.Variables.JsonVariables {
				renderOverrideVar(k, "json", v.Value)
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString(s.Info.Bold(true).Render("  Target Projects (Missing Flag - Will Be Created):") + "\n")
	if len(missingIn) > 0 {
		for _, p := range missingIn {
			sb.WriteString(fmt.Sprintf("    • %s %s\n", s.Warning.Render("➕"), s.Text.Render(fmt.Sprintf("%s (ID: %s)", p.Name, p.ID))))
		}
	} else {
		sb.WriteString("    (None - Flag exists in all projects)\n")
	}
	sb.WriteString("\n")

	if len(mismatches) > 0 {
		sb.WriteString(s.Warning.Bold(true).Render("  Configuration Mismatches (Flag Already Exists but Differs):") + "\n")
		for _, m := range mismatches {
			sb.WriteString(fmt.Sprintf("    • %s (ID: %s):\n", s.Text.Bold(true).Render(m.Project.Name), m.Project.ID))
			for _, d := range m.Differences {
				sb.WriteString(fmt.Sprintf("        %s %s\n", s.Warning.Render("⚠️"), d))
			}
		}
		sb.WriteString("\n")
		sb.WriteString(s.Muted.Render("  Note: Sattchel currently only supports creating missing flags. Aligning differences for existing flags must be done manually.") + "\n\n")
	}

	return sb.String()
}
