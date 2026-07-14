package driving

import (
	"context"
	"fmt"
	"sattchel/internal/optimizely/core"
)

// getProjectCompletions returns formatted shell completion options for Optimizely projects.
func getProjectCompletions(service *core.Service) []string {
	projects, err := service.GetProjects(context.Background())
	if err != nil {
		return nil
	}
	var completions []string
	for _, p := range projects {
		completions = append(completions, fmt.Sprintf("%s\t%s", p.ID, p.Name))
	}
	return completions
}
