package optimizely

import (
	"context"
	"fmt"
	"test-cli/internal/models"
)

type Service interface {
	GetProjects(ctx context.Context) ([]models.Project, error)
	GetConfig(ctx context.Context) (*Configuration, error)
	SetConfig(ctx context.Context, config Configuration) error
	GetFlags(ctx context.Context, projectIDs []string) (map[string][]models.FeatureFlag, error)
}

type service struct {
	config      ConfigDataMapper
	source      *SourceRepository
	flagFactory FlagsDMFactory
}

func NewOptimizelyService(dm ConfigDataMapper, source *SourceRepository, factory FlagsDMFactory) Service {
	return &service{
		config:      dm,
		source:      source,
		flagFactory: factory,
	}
}

func (o service) GetProjects(ctx context.Context) ([]models.Project, error) {
	if o.source == nil {
		return nil, fmt.Errorf("source repository is not initialized")
	}

	cfg, err := o.config.Get(ctx, "")
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

	var results []models.Project
	for _, project := range p {
		if project.Status != nil && project.Status.Valid() {
			if project.Id != nil {
				id := fmt.Sprintf("%d", *project.Id)
				_, active := existingSet[id]
				results = append(results, models.Project{
					ID:       id,
					Name:     project.Name,
					IsActive: active,
					Label:    project.Name,
				})
			}
		}
	}
	return results, nil
}

func (o service) GetConfig(ctx context.Context) (*Configuration, error) {
	return o.config.Get(ctx, "")
}

func (o service) SetConfig(ctx context.Context, config Configuration) error {
	_, err := o.config.Update(ctx, func(value *Configuration) error {
		if config.APIKey != "" {
			value.APIKey = config.APIKey
		}
		if len(config.Projects) > 0 {
			value.Projects = config.Projects
		}
		return nil
	})
	return err
}

func (o service) GetFlags(ctx context.Context, projectIDs []string) (map[string][]models.FeatureFlag, error) {
	result := make(map[string][]models.FeatureFlag)

	for _, pid := range projectIDs {
		dm, err := o.flagFactory.Create(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("failed to create flag mapper for project %s: %w", pid, err)
		}

		flags, err := dm.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get flags for project %s: %w", pid, err)
		}
		result[pid] = flags
	}
	return result, nil
}
