package optimizely

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"test-cli/internal/models"
	"test-cli/internal/optimizely/projects"
)

type environmentDataMapper struct {
	client    *projects.ClientWithResponses
	token     string
	projectID string
}

type EnvironmentDataMapper models.DataMapper[models.Environment]

func BaseV2Client(cfg *Configuration) *projects.ClientWithResponses {
	fc, err := projects.NewClientWithResponses("https://api.optimizely.com/v2", func(client *projects.Client) error {
		if cfg != nil && cfg.APIKey != "" {
			client.RequestEditors = append(client.RequestEditors, WithToken(cfg.APIKey))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	return fc
}

func NewEnvironmentsDM(client *projects.ClientWithResponses, token string, projectID string) (EnvironmentDataMapper, error) {
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

func (e *environmentDataMapper) Get(ctx context.Context, ID string) (*models.Environment, error) {
	err := e.validate()
	if err != nil {
		return nil, err
	}

	id, err := e.getIdForService()
	if err != nil {
		return nil, err
	}

	reporter := models.ProgressFromContext(ctx)
	if reporter != nil {
		reporter.Report(e.projectID, 0.0, "starting")
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

func (e *environmentDataMapper) GetAll(ctx context.Context) ([]models.Environment, error) {
	err := e.validate()
	if err != nil {
		return nil, err
	}

	id, err := e.getIdForService()
	if err != nil {
		return nil, err
	}

	reporter := models.ProgressFromContext(ctx)
	if reporter != nil {
		reporter.Report(e.projectID, 0.0, "starting")
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

	results := make([]models.Environment, 0)
	for _, env := range *response.JSON200 {
		eModel, err := toProjectsEnvironment(env, e.projectID)
		if err != nil {
			slog.Warn("failed to convert an environment")
			continue
		}
		results = append(results, eModel)
	}

	if reporter != nil {
		reporter.Report(e.projectID, 1.0, "done")
	}
	return results, nil
}

func toProjectsEnvironment(env projects.Environment, projectID string) (models.Environment, error) {
	if env.Id == nil {
		return models.Environment{}, fmt.Errorf("missing id")
	}
	id := strconv.Itoa(*env.Id)
	result := models.Environment{
		ID:        id,
		ProjectID: projectID,
		Key:       env.Key,
		Name:      env.Name,
		IsActive:  env.Archived != nil && !*env.Archived,
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

	reporter := models.ProgressFromContext(ctx)
	if reporter != nil {
		reporter.Report(e.projectID, 0.0, "deleting")
	}

	response, err := e.client.DeleteEnvironmentWithResponse(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to delete environment. %v", err)
	}
	if response.StatusCode() != 200 && response.StatusCode() != 204 {
		return "", fmt.Errorf("non-200/204 status code: %d", response.StatusCode())
	}

	if reporter != nil {
		reporter.Report(e.projectID, 1.0, "deleted")
	}
	return ID, nil
}

func (e *environmentDataMapper) Create(ctx context.Context, value models.Environment) (*models.Environment, error) {
	err := e.validate()
	if err != nil {
		return nil, err
	}

	_, err = e.getIdForService()
	if err != nil {
		return nil, err
	}

	reporter := models.ProgressFromContext(ctx)
	if reporter != nil {
		reporter.Report(e.projectID, 0.0, "creating")
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

	if reporter != nil {
		reporter.Report(e.projectID, 1.0, "created")
	}
	return &result, nil
}

func (e *environmentDataMapper) Update(ctx context.Context, updater func(value *models.Environment) error) (*models.Environment, error) {
	err := e.validate()
	if err != nil {
		return nil, err
	}

	id, err := e.getIdForService()
	if err != nil {
		return nil, err
	}

	reporter := models.ProgressFromContext(ctx)
	if reporter != nil {
		reporter.Report(e.projectID, 0.0, "updating")
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

	if reporter != nil {
		reporter.Report(e.projectID, 1.0, "updated")
	}
	return &result, nil
}
