package optimizely

import (
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

func (s SourceRepository) GetProjects() {}

func (s SourceRepository) GetProject() {}
