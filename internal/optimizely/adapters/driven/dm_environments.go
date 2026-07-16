package driven

import (
	"context"
	"fmt"
	"sattchel/internal/optimizely/adapters/driven/projects"
	"sattchel/internal/optimizely/core"
	"strconv"

	"charm.land/log/v2"
)

type environmentDataMapper struct {
	client    *projects.ClientWithResponses
	token     string
	projectID string
}

func BaseV2Client(apiKey string) *projects.ClientWithResponses {
	fc, err := projects.NewClientWithResponses("https://api.optimizely.com/v2", func(client *projects.Client) error {
		if apiKey != "" {
			client.RequestEditors = append(client.RequestEditors, WithToken(apiKey))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return fc
}

func NewEnvironmentsDM(client *projects.ClientWithResponses, token string, projectID string) (core.EnvironmentsRepository, error) {
	return &environmentDataMapper{
		client:    client,
		token:     token,
		projectID: projectID,
	}, nil
}

func (e *environmentDataMapper) validate() error {
	if e.projectID == "" {
		return MissingProjectID
	}
	if e.token == "" {
		return MissingToken
	}
	return nil
}

func (e *environmentDataMapper) getIdForService() (int64, error) {
	id, err := strconv.ParseInt(e.projectID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id format. %v", err)
	}
	return id, nil
}

func (e *environmentDataMapper) Get(ctx context.Context, ID string) (*core.Environment, error) {
	err := e.validate()
	if err != nil {
		return nil, err
	}

	id, err := e.getIdForService()
	if err != nil {
		return nil, err
	}

	response, err := e.client.GetEnvironmentWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment. %v", err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", response.StatusCode())
	}

	if response.JSON200 == nil {
		return nil, fmt.Errorf("missing environment response")
	}

	env, err := toProjectsEnvironment(*response.JSON200, e.projectID)
	if err != nil {
		return nil, err
	}
	return &env, nil
}

func (e *environmentDataMapper) GetAll(ctx context.Context) ([]core.Environment, error) {
	err := e.validate()
	if err != nil {
		return nil, err
	}

	id, err := e.getIdForService()
	if err != nil {
		return nil, err
	}

	// Use pageSize variable defined in dm_flags.go
	response, err := e.client.ListEnvironmentsWithResponse(ctx, &projects.ListEnvironmentsParams{
		PerPage:   new(pageSize),
		ProjectId: id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list environments. %v", err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", response.StatusCode())
	}

	if response.JSON200 == nil {
		return nil, fmt.Errorf("missing environment response")
	}

	results := make([]core.Environment, 0)
	for _, env := range *response.JSON200 {
		eModel, err := toProjectsEnvironment(env, e.projectID)
		if err != nil {
			log.Warn("failed to convert an environment")
			continue
		}
		results = append(results, eModel)
	}

	return results, nil
}

func toProjectsEnvironment(env projects.Environment, projectID string) (core.Environment, error) {
	result := core.Environment{
		ID:        env.Key,
		ProjectID: projectID,
		Key:       env.Key,
		Name:      env.Name,
	}
	if env.Id != nil {
		id := strconv.Itoa(*env.Id)
		result.ID = id
	}
	if env.Archived != nil {
		result.Archived = *env.Archived
	}
	return result, nil
}

func (e *environmentDataMapper) Delete(ctx context.Context, ID string) (string, error) {
	err := e.validate()
	if err != nil {
		return "", err
	}

	id, err := e.getIdForService()
	if err != nil {
		return "", err
	}

	response, err := e.client.DeleteEnvironmentWithResponse(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to delete environment. %v", err)
	}
	if response.StatusCode() != 200 && response.StatusCode() != 204 {
		return "", fmt.Errorf("non-200/204 status code: %d", response.StatusCode())
	}

	return ID, nil
}

func (e *environmentDataMapper) Create(ctx context.Context, value core.Environment) (*core.Environment, error) {
	err := e.validate()
	if err != nil {
		return nil, err
	}

	_, err = e.getIdForService()
	if err != nil {
		return nil, err
	}

	env := projects.Environment{
		Key:  value.Key,
		Name: value.Name,
	}

	response, err := e.client.CreateEnvironmentWithResponse(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment. %v", err)
	}
	if response.StatusCode() != 201 {
		return nil, fmt.Errorf("non-201 status code: %d", response.StatusCode())
	}

	if response.JSON201 == nil {
		return nil, fmt.Errorf("missing environment response")
	}

	result, err := toProjectsEnvironment(*response.JSON201, e.projectID)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (e *environmentDataMapper) Update(ctx context.Context, updater func(value *core.Environment) error) (*core.Environment, error) {
	err := e.validate()
	if err != nil {
		return nil, err
	}

	id, err := e.getIdForService()
	if err != nil {
		return nil, err
	}

	// First get the current state to apply the updater
	getResp, err := e.client.GetEnvironmentWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment for update. %v", err)
	}
	if getResp.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", getResp.StatusCode())
	}
	if getResp.JSON200 == nil {
		return nil, fmt.Errorf("missing environment response")
	}

	currentEnv, err := toProjectsEnvironment(*getResp.JSON200, e.projectID)
	if err != nil {
		return nil, err
	}

	if err := updater(&currentEnv); err != nil {
		return nil, err
	}

	// Convert back to projects.EnvironmentUpdate for the API
	updateBody := projects.EnvironmentUpdate{
		Key:  new(currentEnv.Key),
		Name: new(currentEnv.Name),
	}

	response, err := e.client.UpdateEnvironmentWithResponse(ctx, id, updateBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update environment. %v", err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", response.StatusCode())
	}

	if response.JSON200 == nil {
		return nil, fmt.Errorf("missing environment response")
	}

	result, err := toProjectsEnvironment(*response.JSON200, e.projectID)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
