package optimizely

import (
	"context"
	"fmt"
	"test-cli/internal/repository"
)

type Service interface {
	GetProjects(ctx context.Context) ([]Project, error)
	GetConfig() (*Configuration, error)
	SetConfig(config Configuration) error
}

type service struct {
	config repository.Configuration[Configuration]
	source *SourceRepository
}

func (o service) GetProjects(ctx context.Context) ([]Project, error) {
	if o.source == nil {
		return nil, fmt.Errorf("source repository is not initialized")
	}

	cfg, err := o.config.GetConfig()
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	existingProjects := cfg.Projects
	existingSet := make(map[string]bool, len(existingProjects))
	for _, project := range existingProjects {
		existingSet[project.ID] = true
	}

	p, err := o.source.GetProjects(ctx, cfg.APIKey)
	if err != nil {
		return nil, err
	}

	var results []Project
	for _, project := range p {
		if project.Status != nil && project.Status.Valid() {
			if project.Id != nil {
				id := fmt.Sprintf("%d", *project.Id)
				_, active := existingSet[id]
				results = append(results, Project{
					ID:       id,
					Name:     project.Name,
					IsActive: active,
				})
			}
		}
	}
	return results, nil
}

func (o service) GetConfig() (*Configuration, error) {
	return o.config.GetConfig()
}

func (o service) SetConfig(config Configuration) error {
	c, err := o.config.GetConfig()
	if err != nil {
		return err
	}
	if config.APIKey != "" {
		c.APIKey = config.APIKey
	}

	if len(config.Projects) > 0 {
		c.Projects = config.Projects
	}

	return o.config.SetConfig(*c)
}

func NewOptimizelyService(repo repository.Configuration[Configuration], source *SourceRepository) Service {
	return &service{
		config: repo,
		source: source,
	}
}
