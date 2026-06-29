package driven

import (
	"context"
	"fmt"
	"sattchel/internal/optimizely/adapters/driven/projects"
	"sattchel/internal/optimizely/core"
	"strconv"
)

type projectDataMapper struct {
	client *projects.ClientWithResponses
}

func NewProjectsDM(client *projects.ClientWithResponses) core.ProjectRepository {
	return &projectDataMapper{
		client: client,
	}
}

func (p *projectDataMapper) Get(ctx context.Context, ID string) (*core.Project, error) {
	id, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid project id format: %v", err)
	}

	response, err := p.client.GetProjectWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", response.StatusCode())
	}
	if response.JSON200 == nil {
		return nil, fmt.Errorf("missing project response")
	}

	project, err := toProject(*response.JSON200)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (p *projectDataMapper) GetAll(ctx context.Context) ([]core.Project, error) {
	response, err := p.client.ListProjectsWithResponse(ctx, &projects.ListProjectsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", response.StatusCode())
	}
	if response.JSON200 == nil {
		return nil, fmt.Errorf("missing projects response")
	}

	projs := *response.JSON200
	results := make([]core.Project, 0, len(projs))
	for _, proj := range projs {
		p, err := toProject(proj)
		if err != nil {
			continue
		}
		results = append(results, p)
	}
	return results, nil
}

func toProject(proj projects.Project) (core.Project, error) {
	if proj.Id == nil {
		return core.Project{}, fmt.Errorf("missing project id")
	}

	id := strconv.FormatInt(*proj.Id, 10)
	label := proj.Name
	if proj.Description != nil {
		label = *proj.Description
	}

	return core.Project{
		ID:       id,
		Name:     proj.Name,
		IsActive: false, // set by caller based on config
		Label:    label,
	}, nil
}

func (p *projectDataMapper) Delete(ctx context.Context, ID string) (string, error) {
	return "", fmt.Errorf("delete not supported for projects")
}

func (p *projectDataMapper) Create(ctx context.Context, value core.Project) (*core.Project, error) {
	return nil, fmt.Errorf("create not supported for projects")
}

func (p *projectDataMapper) Update(ctx context.Context, updater func(value *core.Project) error) (*core.Project, error) {
	return nil, fmt.Errorf("update not supported for projects")
}
