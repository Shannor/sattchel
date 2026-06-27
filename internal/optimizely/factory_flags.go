package optimizely

import (
	"context"
	"sattchel/internal/domain"
	"sattchel/internal/optimizely/features"
)

// FlagsDMFactory creates DataMapper instances scoped to a specific project.
type FlagsDMFactory interface {
	Create(ctx context.Context, projectID string) (domain.DataMapper[domain.FeatureFlagDefinition], error)
}

type flagsDMFactory struct {
	client *features.ClientWithResponses
	token  string
}

// NewFlagsDMFactory creates a factory pre-configured with the shared client and token.
func NewFlagsDMFactory(client *features.ClientWithResponses, token string) FlagsDMFactory {
	return &flagsDMFactory{
		client: client,
		token:  token,
	}
}

// Create returns a DataMapper scoped to the given projectID.
func (f *flagsDMFactory) Create(ctx context.Context, projectID string) (domain.DataMapper[domain.FeatureFlagDefinition], error) {
	return NewFlagsDM(f.client, f.token, projectID)
}
