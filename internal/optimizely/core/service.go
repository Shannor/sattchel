package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
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

func (s *Service) SearchFlags(ctx context.Context, projectIDs []string, opts ListFlagsOptions) (map[string][]FeatureFlagDefinition, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := make(map[string][]FeatureFlagDefinition)
	var errs []error

	for _, pid := range projectIDs {
		wg.Add(1)
		go func(projectID string) {
			defer wg.Done()
			dm, err := s.flagFactory.Create(ctx, projectID)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to create flag mapper for project %s: %w", projectID, err))
				mu.Unlock()
				return
			}

			flags, err := dm.GetAll(ctx)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to get flags for project %s: %w", projectID, err))
				mu.Unlock()
				return
			}

			filtered := filterFlags(flags, opts.Query)

			mu.Lock()
			result[projectID] = filtered
			mu.Unlock()
		}(pid)
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, errs[0]
	}

	return result, nil
}

func filterFlags(flags []FeatureFlagDefinition, query string) []FeatureFlagDefinition {
	if query == "" {
		return flags
	}
	query = strings.ToLower(query)
	var filtered []FeatureFlagDefinition
	for _, f := range flags {
		if strings.Contains(strings.ToLower(f.Name), query) ||
			strings.Contains(strings.ToLower(f.Key), query) ||
			strings.Contains(strings.ToLower(f.Description), query) ||
			strings.Contains(strings.ToLower(f.ID), query) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
