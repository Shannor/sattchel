package driving

import (
	"context"
	"sattchel/internal/optimizely/core"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// getProjectCompletions returns formatted shell completion options for Optimizely projects.
func getProjectCompletions(config *Config) []string {
	cfg, err := config.Get()
	if err != nil {
		return nil
	}
	var completions []string
	for _, p := range cfg.Projects {
		completions = append(completions, cobra.CompletionWithDesc(p.ID, p.Label))
	}
	sort.Strings(completions)
	return completions
}

func getFlagKeyCompletions(service *core.Service, config *Config, projectIDs []string, query string) []string {
	cfg, err := config.Get()
	if err != nil || cfg.APIKey == "" {
		return nil
	}
	projectIDs = uniqueNonEmpty(projectIDs)
	if len(projectIDs) == 0 {
		projectIDs = configuredProjectIDs(cfg)
	}
	if len(projectIDs) == 0 {
		return nil
	}

	flagsByProject, err := service.SearchFlags(context.Background(), projectIDs, core.ListFlagsOptions{Query: query})
	if err != nil {
		return nil
	}

	seen := make(map[string]string)
	for _, projectID := range projectIDs {
		for _, flag := range flagsByProject[projectID] {
			if _, ok := seen[flag.Key]; !ok {
				name := strings.TrimSpace(flag.Name)
				if name == "" {
					name = flag.Key
				}
				seen[flag.Key] = name
			}
		}
	}

	keys := make([]string, 0, len(seen))
	for key, name := range seen {
		keys = append(keys, cobra.CompletionWithDesc(key, name))
	}
	sort.Strings(keys)
	return keys
}

func registerProjectFlagCompletion(cmd *cobra.Command, config *Config, flagNames ...string) {
	for _, flagName := range flagNames {
		_ = cmd.RegisterFlagCompletionFunc(flagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return getProjectCompletions(config), cobra.ShellCompDirectiveNoFileComp
		})
	}
}

func registerSourceProjectCompletion(cmd *cobra.Command, config *Config) {
	_ = cmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		results := []string{cobra.CompletionWithDesc("all", "Union mode across target projects")}
		results = append(results, getProjectCompletions(config)...)
		return results, cobra.ShellCompDirectiveNoFileComp
	})
}

func registerFlagKeyCompletion(cmd *cobra.Command, service *core.Service, config *Config, projectFlagName string, flagName string) {
	_ = cmd.RegisterFlagCompletionFunc(flagName, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cfg, err := config.Get()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		projectIDs := make([]string, 0)
		selectedProject, _ := cmd.Flags().GetString(projectFlagName)
		selectedProjects, _ := cmd.Flags().GetStringSlice(projectFlagName)
		if selectedProject != "" {
			if strings.EqualFold(selectedProject, "all") {
				targets, _ := cmd.Flags().GetStringSlice("target")
				projectIDs = append(projectIDs, targets...)
				if len(projectIDs) == 0 {
					projectIDs = append(projectIDs, configuredProjectIDs(cfg)...)
				}
			} else {
				projectIDs = append(projectIDs, selectedProject)
			}
		}
		projectIDs = append(projectIDs, selectedProjects...)
		return getFlagKeyCompletions(service, config, projectIDs, toComplete), cobra.ShellCompDirectiveNoFileComp
	})
}

func registerGetFlagArgCompletion(cmd *cobra.Command, service *core.Service, config *Config, projectFlagName string) {
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		projects, _ := cmd.Flags().GetStringArray(projectFlagName)
		return getFlagKeyCompletions(service, config, projects, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}
