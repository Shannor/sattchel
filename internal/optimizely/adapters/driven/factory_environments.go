package driven

import (
	"context"
	"sattchel/internal/optimizely/adapters/driven/projects"
	"sattchel/internal/optimizely/core"
)

type environmentsDMFactory struct {
	client *projects.ClientWithResponses
	token  string
}

func NewEnvironmentsDMFactory(client *projects.ClientWithResponses, token string) core.EnvironmentsRepositoryFactory {
	return &environmentsDMFactory{
		client: client,
		token:  token,
	}
}

func (f *environmentsDMFactory) Create(ctx context.Context, projectID string) (core.EnvironmentsRepository, error) {
	return NewEnvironmentsDM(f.client, f.token, projectID)
}
