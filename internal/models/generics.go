package models

import "context"

// Configuration Generic actions expected for Configurations in the the system
type Configuration[T any] interface {
	SetConfig(config T) error
	GetConfig() (*T, error)
}

// DataMapper Generic example of all data mapper functions
type DataMapper[T any] interface {
	Get(ctx context.Context, ID string) (*T, error)
	GetAll(ctx context.Context) ([]T, error)
	Update(ctx context.Context, updater func(value *T) error) (*T, error)
	Delete(ctx context.Context, ID string) (string, error)
	Create(ctx context.Context, value T) (*T, error)
}

type FlagApplication interface {
	GetFlag(
		ctx context.Context,
		projectID string,
		environmentIDs []string,
		flagID string,
	) (FeatureFlagDefinition, []FeatureFlagInstance, error)
}
