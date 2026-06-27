package optimizely

import (
	"context"
	"sattchel/internal/models"
	"sattchel/internal/optimizely/projects"
)

// EnvironmentsDMFactory creates DataMapper instances scoped to a specific project.
type EnvironmentsDMFactory interface {
	Create(ctx context.Context, projectID string) (models.DataMapper[models.Environment], error)
}

type environmentsDMFactory struct {
	client *projects.ClientWithResponses
	token  string
}

// NewEnvironmentsDMFactory creates a factory pre-configured with the shared client and token.
func NewEnvironmentsDMFactory(client *projects.ClientWithResponses, token string) EnvironmentsDMFactory {
	return &environmentsDMFactory{
		client: client,
		token:  token,
	}
}

// Create returns a DataMapper scoped to the given projectID.
func (f *environmentsDMFactory) Create(ctx context.Context, projectID string) (models.DataMapper[models.Environment], error) {
	return NewEnvironmentsDM(f.client, f.token, projectID)
}
