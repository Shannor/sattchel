package driving

import (
	"context"
	"fmt"
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
)

func getProjectCompletions(service *core.Service) []string {
	projects, err := service.GetProjects(context.Background())
	if err != nil {
		return nil
	}
	var completions []string
	for _, p := range projects {
		completions = append(completions, fmt.Sprintf("%s\t%s", p.ID, p.Label))
	}
	return completions
}

func getGoalCompletions(service *core.Service, pid string) []string {
	if pid == "" {
		return nil
	}
	goals, err := service.GetGoals(context.Background(), pid)
	if err != nil {
		return nil
	}
	var completions []string
	for _, g := range goals {
		completions = append(completions, fmt.Sprintf("%s\t%s", g.ID, g.Name))
	}
	return completions
}

func getMemberCompletions(service *core.Service) []string {
	members, err := service.GetMembers(context.Background())
	if err != nil {
		return nil
	}
	var completions []string
	for _, m := range members {
		completions = append(completions, fmt.Sprintf("%s\t%s", m.ID, m.Name))
	}
	return completions
}

func getActiveProjectID(cmd *cobra.Command, cfg *Config, projectIDFlag string) string {
	if projectIDFlag != "" {
		return projectIDFlag
	}
	if cmd.Flags().Changed("projectId") {
		if pid, err := cmd.Flags().GetString("projectId"); err == nil && pid != "" {
			return pid
		}
	}
	if lastProj := cfg.CurrentProjectID(); lastProj != "" {
		return lastProj
	}
	return ""
}
