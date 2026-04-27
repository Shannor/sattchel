package optimizely

import (
	"context"
	"fmt"
	"net/http"
	"test-cli/internal/optimizely/features"
	"test-cli/internal/optimizely/projects"
)

type SourceRepository struct {
	featuresClient *features.ClientWithResponses
	projectsClient *projects.ClientWithResponses
}

func NewSourceRepository() (*SourceRepository, error) {
	fc, err := features.NewClientWithResponses("https://api.optimizely.com/")
	if err != nil {
		return nil, err
	}
	pc, err := projects.NewClientWithResponses("https://api.optimizely.com/v2")
	if err != nil {
		return nil, err
	}
	return &SourceRepository{
		featuresClient: fc,
		projectsClient: pc,
	}, nil
}

// WithToken returns a RequestEditorFn that injects the auth header
func WithToken(token string) projects.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
	}
}

func (s SourceRepository) GetProjects(ctx context.Context, token string) ([]projects.Project, error) {

	response, err := s.projectsClient.ListProjectsWithResponse(
		ctx,
		&projects.ListProjectsParams{},
		WithToken(token),
	)

	if err != nil {
		return nil, err
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", response.StatusCode())
	}
	if response.JSON401 != nil {
		return nil, fmt.Errorf("unauthorized: %v", response.JSON401.Message)
	}
	if response.JSON200 != nil {
		return *response.JSON200, nil
	}
	return nil, fmt.Errorf("no projects found in response")
}

func (s SourceRepository) GetProject() {}
