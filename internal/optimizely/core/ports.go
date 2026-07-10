package core

import "context"

// Repository defines generic CRUD/mapper interfaces for repositories
type Repository[T any] interface {
	Get(ctx context.Context, ID string) (*T, error)
	GetAll(ctx context.Context) ([]T, error)
	Update(ctx context.Context, updater func(value *T) error) (*T, error)
	Delete(ctx context.Context, ID string) (string, error)
	Create(ctx context.Context, value T) (*T, error)
}

type ProjectRepository Repository[Project]
type ListFlagsOptions struct {
	Query string
}

type FlagsRepository interface {
	Repository[FeatureFlagDefinition]
}
type EnvironmentsRepository Repository[Environment]

type FlagsRepositoryFactory interface {
	Create(ctx context.Context, projectID string) (FlagsRepository, error)
}

type EnvironmentsRepositoryFactory interface {
	Create(ctx context.Context, projectID string) (EnvironmentsRepository, error)
}
