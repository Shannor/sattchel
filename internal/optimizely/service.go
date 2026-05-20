package optimizely

import (
	"context"
	"fmt"
	"log/slog"
	"test-cli/internal/models"
	"test-cli/internal/set"
)

type Service interface {
	GetAllProjects(ctx context.Context) ([]models.Project, error)
	GetSelectedProjects(ctx context.Context) ([]models.Project, error)
	GetConfig(ctx context.Context) (*Configuration, error)
	SetConfig(ctx context.Context, config Configuration) error

	GetFlag(ctx context.Context, projectID string, environmentIDs []string, flagID string) (*models.FeatureFlagDefinition, []models.FeatureFlagInstance, error)
	GetFlags(ctx context.Context, projectIDs []string) (map[string][]models.FeatureFlagDefinition, error)
}

type service struct {
	config      ConfigDataMapper
	projectDM   ProjectDataMapper
	flagFactory FlagsDMFactory
	envFactory  EnvironmentsDMFactory
}

func NewOptimizelyService(
	dm ConfigDataMapper,
	projectDM ProjectDataMapper,
	factory FlagsDMFactory,
	dmFactory EnvironmentsDMFactory,
) Service {
	return &service{
		config:      dm,
		projectDM:   projectDM,
		flagFactory: factory,
		envFactory:  dmFactory,
	}
}

func (o service) GetSelectedProjects(ctx context.Context) ([]models.Project, error) {
	cfg, err := o.config.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	return cfg.Projects, nil
}

func (o service) GetAllProjects(ctx context.Context) ([]models.Project, error) {
	cfg, err := o.config.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	existingProjects := cfg.Projects
	existing := set.NewFromFunc[string](existingProjects, func(v models.Project) string {
		return v.ID
	})

	flagsProjects, err := o.projectDM.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	var results []models.Project
	for _, proj := range flagsProjects {
		id := proj.ID
		if existing.Contains(id) {
			results = append(results, models.Project{
				ID:       id,
				Name:     proj.Name,
				IsActive: true,
				Label:    proj.Label,
			})
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

func (o service) GetFlags(ctx context.Context, projectIDs []string) (map[string][]models.FeatureFlagDefinition, error) {
	result := make(map[string][]models.FeatureFlagDefinition)

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

func (o service) GetFlag(ctx context.Context, projectID string, environmentIDs []string, flagID string) (*models.FeatureFlagDefinition, []models.FeatureFlagInstance, error) {
	instances := make([]models.FeatureFlagInstance, 0)
	dm, err := o.flagFactory.Create(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create flag mapper for project %s: %w", projectID, err)
	}
	flag, err := dm.Get(ctx, flagID)
	if err != nil {
		return nil, nil, err
	}

	if len(environmentIDs) == 0 {
		r, err := flag.AllInstances()
		if err != nil {
			return nil, nil, err
		}
		instances = append(instances, r...)
	}

	for _, environment := range environmentIDs {
		i, err := flag.ByEnvID(environment)
		if err != nil {
			slog.Error(
				"error getting flag instance",
				slog.String("flagId", flagID),
				slog.String("envId", environment),
			)
			continue
		}
		instances = append(instances, *i)
	}

	return flag, instances, nil
}
