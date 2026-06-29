package core

import (
	"context"
	"fmt"
	"log/slog"
	"sattchel/pkg/set"
)

type Service struct {
	config      ConfigRepository
	projectRepo ProjectRepository
	flagFactory FlagsRepositoryFactory
	envFactory  EnvironmentsRepositoryFactory
}

func NewService(
	config ConfigRepository,
	projectRepo ProjectRepository,
	flagFactory FlagsRepositoryFactory,
	envFactory EnvironmentsRepositoryFactory,
) *Service {
	return &Service{
		config:      config,
		projectRepo: projectRepo,
		flagFactory: flagFactory,
		envFactory:  envFactory,
	}
}

func (s *Service) GetSelectedProjects(ctx context.Context) ([]Project, error) {
	cfg, err := s.config.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	return cfg.Projects, nil
}

func (s *Service) GetAllProjects(ctx context.Context) ([]Project, error) {
	cfg, err := s.config.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	existingProjects := cfg.Projects
	existing := set.NewFromFunc[string](existingProjects, func(v Project) string {
		return v.ID
	})

	flagsProjects, err := s.projectRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	var results []Project
	for _, proj := range flagsProjects {
		id := proj.ID
		if existing.Contains(id) {
			results = append(results, Project{
				ID:       id,
				Name:     proj.Name,
				IsActive: true,
				Label:    proj.Label,
			})
		}
	}
	return results, nil
}

func (s *Service) GetConfig(ctx context.Context) (*Configuration, error) {
	return s.config.Get(ctx, "")
}

func (s *Service) SetConfig(ctx context.Context, config Configuration) error {
	_, err := s.config.Update(ctx, func(value *Configuration) error {
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

func (s *Service) GetFlags(ctx context.Context, projectIDs []string) (map[string][]FeatureFlagDefinition, error) {
	result := make(map[string][]FeatureFlagDefinition)

	for _, pid := range projectIDs {
		dm, err := s.flagFactory.Create(ctx, pid)
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

func (s *Service) GetFlag(ctx context.Context, projectID string, environmentIDs []string, flagID string) (*FeatureFlagDefinition, []FeatureFlagInstance, error) {
	instances := make([]FeatureFlagInstance, 0)
	dm, err := s.flagFactory.Create(ctx, projectID)
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
