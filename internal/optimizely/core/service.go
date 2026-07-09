package core

import (
	"context"
	"fmt"
	"log/slog"
)

type Service struct {
	projectRepo ProjectRepository
	flagFactory FlagsRepositoryFactory
	envFactory  EnvironmentsRepositoryFactory
}

func NewService(
	projectRepo ProjectRepository,
	flagFactory FlagsRepositoryFactory,
	envFactory EnvironmentsRepositoryFactory,
) *Service {
	return &Service{
		projectRepo: projectRepo,
		flagFactory: flagFactory,
		envFactory:  envFactory,
	}
}

func (s *Service) GetProjects(ctx context.Context) ([]Project, error) {
	return s.projectRepo.GetAll(ctx)
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
