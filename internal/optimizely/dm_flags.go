package optimizely

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"test-cli/internal/models"
	"test-cli/internal/optimizely/features"
)

var (
	MissingToken     = errors.New("token required")
	MissingProjectID = errors.New("project ID required")
)

type flagDataMapper struct {
	client    *features.ClientWithResponses
	token     string
	projectID string
}

type FlagDataMapper models.DataMapper[models.FeatureFlag]

func BaseFlagClient(cfg *Configuration) *features.ClientWithResponses {
	fc, err := features.NewClientWithResponses("https://api.optimizely.com/flags/v1/", func(client *features.Client) error {
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

func NewFlagsDM(client *features.ClientWithResponses, token string, projectID string) (FlagDataMapper, error) {

	return &flagDataMapper{
		client:    client,
		token:     token,
		projectID: projectID,
	}, nil
}

func (f *flagDataMapper) Get(ctx context.Context, ID string) (*models.FeatureFlag, error) {
	err := f.validate()
	if err != nil {
		return nil, err
	}

	id, err := f.getIdForService()
	if err != nil {
		return nil, err
	}
	reporter := models.ProgressFromContext(ctx)
	if reporter != nil {
		reporter.Report(f.projectID, 0.0, "starting")
	}
	response, err := f.client.FetchFlagWithResponse(ctx, id, ID)
	if err != nil {
		return nil, err
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", response.StatusCode())
	}

	if response.JSON200 == nil {
		return nil, fmt.Errorf("missing flag response")
	}

	result, err := toFeatureFlag(*response.JSON200)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (f *flagDataMapper) validate() error {
	if f.projectID == "" {
		return MissingProjectID
	}
	if f.token == "" {
		return MissingToken
	}
	return nil
}

func (f *flagDataMapper) getIdForService() (int, error) {
	id, err := strconv.Atoi(f.projectID)
	if err != nil {
		return 0, fmt.Errorf("invalid id format. %v", err)
	}
	return id, nil
}

const (
	pageSize = 20
)

func (f *flagDataMapper) GetAll(ctx context.Context) ([]models.FeatureFlag, error) {
	err := f.validate()
	if err != nil {
		return nil, err
	}

	id, err := f.getIdForService()
	if err != nil {
		return nil, err
	}

	reporter := models.ProgressFromContext(ctx)
	if reporter != nil {
		reporter.Report(f.projectID, 0.0, "starting")
	}
	response, err := f.client.ListFlagsWithResponse(ctx, id, &features.ListFlagsParams{
		PageWindow: new(pageSize),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list flags. %v", err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code: %d", response.StatusCode())
	}

	if response.JSON200 == nil {
		return nil, fmt.Errorf("missing flag response")
	}

	info := response.JSON200

	results := make([]models.FeatureFlag, 0)
	for _, flag := range info.Items {

		ff, err := toFeatureFlag(flag)
		if err != nil {
			slog.Warn("failed to convert a feature flag")
			continue
		}
		results = append(results, ff)
	}

	if info.NextUrl == nil {
		if reporter != nil {
			reporter.Report(f.projectID, 1.0, "done")
		}
		return results, nil
	}

	// We can do this only because we're sending the page size we want.
	// If we didn't send a pageSize/PageWindow it wouldn't work this way. The API would only send one url in the array
	tokens := make([]string, 0)
	for _, u := range *info.NextUrl {
		tokens = append(tokens, extractPageToken(u))
	}

	var wg sync.WaitGroup
	var completedPages atomic.Int64
	totalPages := info.TotalPages
	for _, token := range tokens {
		wg.Go(func() {
			response, err := f.client.ListFlagsWithResponse(ctx, id, &features.ListFlagsParams{
				PageToken:  &token,
				PageWindow: new(pageSize),
			})
			if err != nil {
				slog.Error("failed to get flags", slog.String("error", err.Error()))
				return
			}
			if response.StatusCode() != 200 {
				slog.Error("non-200 status code", slog.Int("code", response.StatusCode()))
				return
			}

			if response.JSON200 == nil {
				slog.Error("missing flag response")
				return
			}

			r := response.JSON200
			for _, flag := range r.Items {
				ff, err := toFeatureFlag(flag)
				if err != nil {
					slog.Warn("failed to convert a feature flag")
					continue
				}
				results = append(results, ff)
			}
			n := completedPages.Add(1)
			if reporter != nil {
				reporter.Report(f.projectID, float64(n)/float64(totalPages), "fetching pages")
			}
		})
	}

	wg.Wait()
	return results, nil
}

func getVariations(client *features.ClientWithResponses, projectID string, flagID string) {
	client.ListVariationsWithResponse()

}

// extractPageToken pulls the page_token query param from a next_url value.
// The API returns full URLs like "/projects/.../flags?page_token=...&page_window=20"
// but the PageToken param expects only the token string.
func extractPageToken(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("page_token")
}

func toFeatureFlag(flag features.Flag) (models.FeatureFlag, error) {
	if flag.Id == nil {
		return models.FeatureFlag{}, fmt.Errorf("missing id")
	}
	id := strconv.Itoa(*flag.Id)
	// TODO: need to add support for the default Variables and the variation variables
	result := models.FeatureFlag{
		ID:               id,
		Name:             flag.Name,
		DefaultVariables: models.Variables{},
	}

	if flag.Archived != nil {
		result.IsArchived = *flag.Archived
	}

	if flag.Environments != nil {
		envs := make([]models.Environment, 0)
		for _, environment := range *flag.Environments {
			e, err := toEnvironment(environment)
			if err != nil {
				slog.Error("Failed to convert an environment", slog.String("err", err.Error()))
				continue
			}
			envs = append(envs, e)
		}
		result.Environments = envs
	}
	return result, nil
}

func toEnvironment(env features.FlagEnvironment) (models.Environment, error) {
	if env.Id == nil {
		return models.Environment{}, fmt.Errorf("missing id")
	}
	id := strconv.Itoa(int(*env.Id))
	result := models.Environment{
		ID:   id,
		Key:  env.Key,
		Name: env.Name,
	}
	if env.Enabled != nil {
		result.Enabled = *env.Enabled
	}
	return result, nil
}

func (f *flagDataMapper) Delete(ctx context.Context, ID string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (f *flagDataMapper) Create(ctx context.Context, value models.FeatureFlag) (*models.FeatureFlag, error) {
	//TODO implement me
	panic("implement me")
}

func (f *flagDataMapper) Update(ctx context.Context, updater func(value *models.FeatureFlag) error) (*models.FeatureFlag, error) {
	//TODO implement me
	panic("implement me")
}
