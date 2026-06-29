package driven

import (
	"context"
	"sattchel/internal/optimizely/adapters/driven/features"
	"sattchel/internal/optimizely/core"
)

type flagsDMFactory struct {
	client *features.ClientWithResponses
	token  string
}

func NewFlagsDMFactory(client *features.ClientWithResponses, token string) core.FlagsRepositoryFactory {
	return &flagsDMFactory{
		client: client,
		token:  token,
	}
}

func (f *flagsDMFactory) Create(ctx context.Context, projectID string) (core.FlagsRepository, error) {
	return NewFlagsDM(f.client, f.token, projectID)
}
