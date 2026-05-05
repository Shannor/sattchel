package optimizely

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
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
	//TODO implement me
	panic("implement me")
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

	// Technically, the API says that it will return 1 page.
	// But when passing pageWindow it returns a list of all the next URLs needed after the first request.
	// Could be an optimization later
	if info.NextUrl != nil && len(*info.NextUrl) > 0 {
		nextToken := extractPageToken((*info.NextUrl)[0])
		for nextToken != "" {
			response, err := f.client.ListFlagsWithResponse(ctx, id, &features.ListFlagsParams{
				PageToken:  &nextToken,
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

			r := response.JSON200
			for _, flag := range r.Items {
				ff, err := toFeatureFlag(flag)
				if err != nil {
					slog.Warn("failed to convert a feature flag")
					continue
				}
				results = append(results, ff)
			}

			if r.NextUrl != nil && len(*r.NextUrl) > 0 {
				nextToken = extractPageToken((*r.NextUrl)[0])
			} else {
				nextToken = ""
			}
		}
	}

	return results, nil
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
	return models.FeatureFlag{
		ID:        id,
		Name:      flag.Name,
		Variables: models.Variables{},
	}, nil
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

func (f *flagDataMapper) With(ctx context.Context, updater func(dm *models.DataMapper[models.FeatureFlag]) error) {
	//TODO implement me
	panic("implement me")
}
