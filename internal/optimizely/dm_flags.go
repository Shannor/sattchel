package optimizely

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sattchel/internal/models"
	"sattchel/internal/optimizely/features"
	"sattchel/internal/optimizely/projects"
	"strconv"
	"sync"
	"sync/atomic"
)

var (
	MissingToken     = errors.New("token required")
	MissingProjectID = errors.New("project ID required")
)

const (
	largePageSize = int64(100)
	pageSize      = 20
)

type flagDataMapper struct {
	client    *features.ClientWithResponses
	token     string
	projectID string
}

type FlagDataMapper models.DataMapper[models.FeatureFlagDefinition]

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

func BaseEnvironmentClient(cfg *Configuration) *projects.ClientWithResponses {
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

func (f *flagDataMapper) Get(ctx context.Context, ID string) (*models.FeatureFlagDefinition, error) {
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

	enriched, err := f.enrichFlag(ctx, response.JSON200)
	if err != nil {
		return nil, err
	}
	return enriched, nil
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

func (f *flagDataMapper) GetAll(ctx context.Context) ([]models.FeatureFlagDefinition, error) {
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

	results := make([]models.FeatureFlagDefinition, 0)
	for _, flag := range info.Items {
		enriched, err := f.enrichFlag(ctx, &flag)
		if err != nil {
			slog.Warn("failed to enrich a feature flag", slog.String("flag_key", flag.Key))
			continue
		}
		results = append(results, *enriched)
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
				enriched, err := f.enrichFlag(ctx, &flag)
				if err != nil {
					slog.Warn("failed to enrich a feature flag", slog.String("flag_key", flag.Key))
					continue
				}
				results = append(results, *enriched)
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

func optionalString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func extractPageToken(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("page_token")
}

func toFeatureFlag(flag features.Flag) (models.FeatureFlagDefinition, error) {
	id := flag.Key
	if flag.Id != nil {
		id = strconv.Itoa(*flag.Id)
	}
	result := models.FeatureFlagDefinition{
		ID:               id,
		Key:              flag.Key,
		Name:             flag.Name,
		DefaultVariables: parseVariableDefinitions(flag.VariableDefinitions),
		Description:      optionalString(flag.Description),
		CreatedAt:        flag.CreatedTime,
		CreatedBy:        flag.CreatedByUserEmail,
	}

	if flag.Archived != nil {
		result.Archived = *flag.Archived
	}

	targets := make([]models.Target, 0)
	if flag.Environments != nil {
		for _, environment := range *flag.Environments {
			t, err := toTarget(environment)
			if err != nil {
				slog.Error("Failed to convert an environment", slog.String("err", err.Error()))
				continue
			}
			targets = append(targets, t)
		}
	}
	result.SetTargets(targets)
	return result, nil
}

func toOverride(variation features.Variation) models.Override {
	override := models.Override{
		ID:          variation.Key,
		Key:         variation.Key,
		Name:        variation.Name,
		Description: optionalString(variation.Description),
	}
	override.Variables = fromVariableValue(variation.Variables)
	return override
}

func toTarget(env features.FlagEnvironment) (models.Target, error) {
	result := models.Target{
		EnvironmentID: env.Key,
		OverrideID:    optionalString(env.DefaultVariationKey),
	}
	if env.Enabled != nil {
		result.IsEnabled = *env.Enabled
	}
	return result, nil
}

func toEnvironment(env features.FlagEnvironment) (models.Environment, error) {
	result := models.Environment{
		ID:   env.Key,
		Key:  env.Key,
		Name: env.Name,
	}
	if env.Id != nil {
		id := strconv.Itoa(int(*env.Id))
		result.ID = id
	}
	return result, nil
}

func fromVariableValue(defs *map[string]features.VariableValue) models.Variables {
	vars := models.Variables{}
	if defs == nil {
		return vars
	}

	for key, def := range *defs {
		if def.Type == nil {
			continue
		}
		switch *def.Type {
		case features.VariableValueTypeBoolean:
			b, err := strconv.ParseBool(def.Value)
			if err == nil {
				if vars.BoolVariables == nil {
					vars.BoolVariables = make(models.VariableMap[bool])
				}
				vars.BoolVariables[key] = models.Variable[bool]{
					Key:   key,
					Value: b,
					Type:  string(*def.Type),
				}
			}

		case features.VariableValueTypeInteger:
			n, err := strconv.ParseInt(def.Value, 10, 64)
			if err == nil {
				if vars.IntVariables == nil {
					vars.IntVariables = make(models.VariableMap[int])
				}
				vars.IntVariables[key] = models.Variable[int]{
					Key:   key,
					Value: int(n),
					Type:  string(*def.Type),
				}
			}

		case features.VariableValueTypeDouble:
			f, err := strconv.ParseFloat(def.Value, 64)
			if err == nil {
				if vars.FloatVariables == nil {
					vars.FloatVariables = make(models.VariableMap[float64])
				}
				vars.FloatVariables[key] = models.Variable[float64]{
					Key:   key,
					Value: f,
					Type:  string(*def.Type),
				}
			}

		case features.VariableValueTypeJson:
			if vars.JsonVariables == nil {
				vars.JsonVariables = make(models.VariableMap[any])
			}
			vars.JsonVariables[key] = models.Variable[any]{
				Key:   key,
				Value: def.Value,
				Type:  string(*def.Type),
			}

		case features.VariableValueTypeString:
			if vars.StringVariables == nil {
				vars.StringVariables = make(models.VariableMap[string])
			}
			vars.StringVariables[key] = models.Variable[string]{
				Key:   key,
				Value: def.Value,
				Type:  string(*def.Type),
			}
		}
	}

	return vars
}

func parseVariableDefinitions(defs *map[string]features.VariableDefinition) models.Variables {
	vars := models.Variables{}
	if defs == nil {
		return vars
	}

	for key, def := range *defs {
		switch def.Type {
		case features.VariableDefinitionTypeBoolean:
			b, err := strconv.ParseBool(def.DefaultValue)
			if err == nil {
				if vars.BoolVariables == nil {
					vars.BoolVariables = make(models.VariableMap[bool])
				}
				vars.BoolVariables[key] = models.Variable[bool]{
					Key:         key,
					Value:       b,
					Type:        string(def.Type),
					Description: optionalString(def.Description),
				}
			}

		case features.VariableDefinitionTypeInteger:
			n, err := strconv.ParseInt(def.DefaultValue, 10, 64)
			if err == nil {
				if vars.IntVariables == nil {
					vars.IntVariables = make(models.VariableMap[int])
				}
				vars.IntVariables[key] = models.Variable[int]{
					Key:         key,
					Value:       int(n),
					Type:        string(def.Type),
					Description: optionalString(def.Description),
				}
			}

		case features.VariableDefinitionTypeDouble:
			f, err := strconv.ParseFloat(def.DefaultValue, 64)
			if err == nil {
				if vars.FloatVariables == nil {
					vars.FloatVariables = make(models.VariableMap[float64])
				}
				vars.FloatVariables[key] = models.Variable[float64]{
					Key:         key,
					Value:       f,
					Type:        string(def.Type),
					Description: optionalString(def.Description),
				}
			}

		case features.VariableDefinitionTypeJson:
			if vars.JsonVariables == nil {
				vars.JsonVariables = make(models.VariableMap[any])
			}
			vars.JsonVariables[key] = models.Variable[any]{
				Key:         key,
				Value:       def.DefaultValue,
				Type:        string(def.Type),
				Description: optionalString(def.Description),
			}

		case features.VariableDefinitionTypeString:
			if vars.StringVariables == nil {
				vars.StringVariables = make(models.VariableMap[string])
			}
			vars.StringVariables[key] = models.Variable[string]{
				Key:         key,
				Value:       def.DefaultValue,
				Type:        string(def.Type),
				Description: optionalString(def.Description),
			}
		}
	}

	return vars
}

// fetchAllVariableDefinitions retrieves all variable definitions for a flag,
// using pagination if there are more than pageSize items.
func (f *flagDataMapper) fetchAllVariableDefinitions(ctx context.Context, flagKey string) (*map[string]features.VariableDefinition, error) {
	projectId, err := strconv.Atoi(f.projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID %q: %w", f.projectID, err)
	}
	response, err := f.client.ListVariableDefinitionsWithResponse(ctx, features.ProjectId(projectId), flagKey, &features.ListVariableDefinitionsParams{
		PerPage:    new(int64(100)),
		PageWindow: new(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list variable definitions for flag %s: %w", flagKey, err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code listing variable definitions for flag %s: %d", flagKey, response.StatusCode())
	}
	if response.JSON200 == nil || response.JSON200.Items == nil {
		return &map[string]features.VariableDefinition{}, nil
	}

	defs := make(map[string]features.VariableDefinition)
	for _, v := range *response.JSON200.Items {
		defs[v.Key] = v
	}

	// Handle pagination if there are more pages
	if response.JSON200.NextUrl != nil {
		tokens := make([]string, 0)
		for _, u := range *response.JSON200.NextUrl {
			tokens = append(tokens, extractPageToken(u))
		}
		for _, token := range tokens {
			pageResp, err := f.client.ListVariableDefinitionsWithResponse(ctx, features.ProjectId(projectId), flagKey, &features.ListVariableDefinitionsParams{
				PageToken:  &token,
				PerPage:    new(largePageSize),
				PageWindow: new(1),
			})
			if err != nil {
				slog.Warn("failed to fetch next page of variable definitions", slog.String("flag", flagKey), slog.String("error", err.Error()))
				continue
			}
			if pageResp.StatusCode() != 200 || pageResp.JSON200 == nil || pageResp.JSON200.Items == nil {
				continue
			}
			for _, v := range *pageResp.JSON200.Items {
				defs[v.Key] = v
			}
		}
	}

	return &defs, nil
}

// fetchAllVariations retrieves all variations for a flag,
// using pagination if there are more than pageSize items.
func (f *flagDataMapper) fetchAllVariations(ctx context.Context, flagKey string) ([]features.Variation, error) {
	projectId, err := strconv.Atoi(f.projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID %q: %w", f.projectID, err)
	}
	response, err := f.client.ListVariationsWithResponse(ctx, features.ProjectId(projectId), flagKey, &features.ListVariationsParams{
		PerPage:    new(largePageSize),
		PageWindow: new(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list variations for flag %s: %w", flagKey, err)
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("non-200 status code listing variations for flag %s: %d", flagKey, response.StatusCode())
	}
	if response.JSON200 == nil || response.JSON200.Items == nil {
		return []features.Variation{}, nil
	}

	variations := make([]features.Variation, 0, len(response.JSON200.Items))
	for _, v := range response.JSON200.Items {
		variations = append(variations, v)
	}

	// Handle pagination if there are more pages
	if response.JSON200.NextUrl != nil {
		tokens := make([]string, 0)
		for _, u := range *response.JSON200.NextUrl {
			tokens = append(tokens, extractPageToken(u))
		}
		for _, token := range tokens {
			pageResp, err := f.client.ListVariationsWithResponse(ctx, features.ProjectId(projectId), flagKey, &features.ListVariationsParams{
				PageToken:  &token,
				PerPage:    new(largePageSize),
				PageWindow: new(1),
			})
			if err != nil {
				slog.Warn("failed to fetch next page of variations", slog.String("flag", flagKey), slog.String("error", err.Error()))
				continue
			}
			if pageResp.StatusCode() != 200 || pageResp.JSON200 == nil || pageResp.JSON200.Items == nil {
				continue
			}
			for _, v := range pageResp.JSON200.Items {
				variations = append(variations, v)
			}
		}
	}

	return variations, nil
}

// enrichFlag fetches additional data (variable definitions and variations) for a flag
// and returns the enriched FeatureFlagDefinition.
func (f *flagDataMapper) enrichFlag(ctx context.Context, flag *features.Flag) (*models.FeatureFlagDefinition, error) {
	flag.VariableDefinitions = new(getAllDefinitions(ctx, flag, f))
	// Fetch all variations
	allVariations, err := f.fetchAllVariations(ctx, flag.Key)
	if err != nil {
		slog.Warn("failed to fetch all variations", slog.String("flag", flag.Key), slog.String("error", err.Error()))
	}

	result, err := toFeatureFlag(*flag)
	if err != nil {
		return nil, err
	}
	overrides := make([]models.Override, 0)
	for _, variation := range allVariations {
		overrides = append(overrides, toOverride(variation))
	}
	result.SetOverrides(overrides)
	return &result, nil
}

// TODO: Convert this to just a standard function to clean it up
func getAllDefinitions(ctx context.Context, flag *features.Flag, f *flagDataMapper) map[string]features.VariableDefinition {
	// TODO: Split into it's own function
	definitions := make(map[string]features.VariableDefinition)
	// Fetch all variable definitions if the flag has more than 5
	if flag.VariableDefinitions != nil {
		definitions = *flag.VariableDefinitions
		if len(definitions) > 5 {
			allDefs, err := f.fetchAllVariableDefinitions(ctx, flag.Key)
			if err != nil {
				slog.Warn("failed to fetch all variable definitions", slog.String("flag", flag.Key), slog.String("error", err.Error()))
			} else {
				// Merge fetched definitions into the flag's existing definitions
				if allDefs != nil {
					for k, v := range *allDefs {
						definitions[k] = v
					}
				}
			}
		}
	}
	return definitions
}

func (f *flagDataMapper) Delete(ctx context.Context, ID string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (f *flagDataMapper) Create(ctx context.Context, value models.FeatureFlagDefinition) (*models.FeatureFlagDefinition, error) {
	//TODO implement me
	panic("implement me")
}

func (f *flagDataMapper) Update(ctx context.Context, updater func(value *models.FeatureFlagDefinition) error) (*models.FeatureFlagDefinition, error) {
	//TODO implement me
	panic("implement me")
}
