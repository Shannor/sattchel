package driven

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sattchel/internal/optimizely/adapters/driven/features"
	"sattchel/internal/optimizely/core"
	"slices"
	"strconv"
	"sync"

	"charm.land/log/v2"
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

// WithToken returns a RequestEditorFn that injects the auth header.
func WithToken(token string) func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
	}
}

func BaseFlagClient(apiKey string) *features.ClientWithResponses {
	fc, err := features.NewClientWithResponses("https://api.optimizely.com/flags/v1/", func(client *features.Client) error {
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

func NewFlagsDM(client *features.ClientWithResponses, token string, projectID string) (core.FlagsRepository, error) {
	return &flagDataMapper{
		client:    client,
		token:     token,
		projectID: projectID,
	}, nil
}

func (f *flagDataMapper) Get(ctx context.Context, ID string) (*core.FeatureFlagDefinition, error) {
	err := f.validate()
	if err != nil {
		return nil, err
	}

	id, err := f.getIdForService()
	if err != nil {
		return nil, err
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

func (f *flagDataMapper) GetAll(ctx context.Context) ([]core.FeatureFlagDefinition, error) {
	err := f.validate()
	if err != nil {
		return nil, err
	}

	id, err := f.getIdForService()
	if err != nil {
		return nil, err
	}

	perPageVal := largePageSize
	params := &features.ListFlagsParams{
		PerPage:    &perPageVal,
		PageWindow: new(pageSize),
	}

	response, err := f.client.ListFlagsWithResponse(ctx, id, params)
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

	var mu sync.Mutex
	results := make([]core.FeatureFlagDefinition, 0)
	for _, flag := range info.Items {
		fDef, err := toFeatureFlag(flag)
		if err != nil {
			log.Warn("failed to convert a feature flag", "flag_key", flag.Key)
			continue
		}
		results = append(results, fDef)
	}

	if info.NextUrl == nil {
		return results, nil
	}

	// We can do this only because we're sending the page size we want.
	// If we didn't send a pageSize/PageWindow it wouldn't work this way. The API would only send one url in the array
	tokens := make([]string, 0)
	for _, u := range *info.NextUrl {
		tokens = append(tokens, extractPageToken(u))
	}

	var wg sync.WaitGroup
	for _, token := range tokens {
		tok := token
		wg.Go(func() {
			pageParams := &features.ListFlagsParams{
				PageToken:  &tok,
				PerPage:    &perPageVal,
				PageWindow: new(pageSize),
			}
			response, err := f.client.ListFlagsWithResponse(ctx, id, pageParams)
			if err != nil {
				log.Error("failed to get flags", "error", err.Error())
				return
			}
			if response.StatusCode() != 200 {
				log.Error("non-200 status code", "code", response.StatusCode())
				return
			}

			if response.JSON200 == nil {
				log.Error("missing flag response")
				return
			}

			r := response.JSON200
			for _, flag := range r.Items {
				fDef, err := toFeatureFlag(flag)
				if err != nil {
					log.Warn("failed to convert a feature flag", "flag_key", flag.Key)
					continue
				}
				mu.Lock()
				results = append(results, fDef)
				mu.Unlock()
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

func toFeatureFlag(flag features.Flag) (core.FeatureFlagDefinition, error) {
	id := flag.Key
	if flag.Id != nil {
		id = strconv.Itoa(*flag.Id)
	}
	result := core.FeatureFlagDefinition{
		ID:               id,
		Key:              flag.Key,
		Name:             flag.Name,
		DefaultVariables: parseVariableDefinitions(flag.VariableDefinitions),
		Description:      optionalString(flag.Description),
		SourceURL:        optionalString(flag.Url),
		CreatedAt:        flag.CreatedTime,
		CreatedBy:        flag.CreatedByUserEmail,
	}

	if flag.Archived != nil {
		result.Archived = *flag.Archived
	}

	targets := make([]core.Target, 0)
	if flag.Environments != nil {
		for _, environment := range *flag.Environments {
			t, err := toTarget(environment)
			if err != nil {
				log.Error("Failed to convert an environment", "err", err.Error())
				continue
			}
			targets = append(targets, t)
		}
	}
	result.Targets = targets
	return result, nil
}

func toOverride(variation features.Variation) core.Override {
	override := core.Override{
		ID:          variation.Key,
		Key:         variation.Key,
		Name:        variation.Name,
		Description: optionalString(variation.Description),
	}
	override.Variables = fromVariableValue(variation.Variables)
	return override
}

func toTarget(env features.FlagEnvironment) (core.Target, error) {
	result := core.Target{
		EnvironmentID: env.Key,
		OverrideID:    optionalString(env.DefaultVariationKey),
	}
	if env.Enabled != nil {
		result.IsEnabled = *env.Enabled
	}
	return result, nil
}

func toEnvironment(env features.FlagEnvironment) (core.Environment, error) {
	result := core.Environment{
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

func fromVariableValue(defs *map[string]features.VariableValue) core.Variables {
	vars := core.Variables{}
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
					vars.BoolVariables = make(core.VariableMap[bool])
				}
				vars.BoolVariables[key] = core.Variable[bool]{
					Key:   key,
					Value: b,
					Type:  string(*def.Type),
				}
			}

		case features.VariableValueTypeInteger:
			n, err := strconv.ParseInt(def.Value, 10, 64)
			if err == nil {
				if vars.IntVariables == nil {
					vars.IntVariables = make(core.VariableMap[int])
				}
				vars.IntVariables[key] = core.Variable[int]{
					Key:   key,
					Value: int(n),
					Type:  string(*def.Type),
				}
			}

		case features.VariableValueTypeDouble:
			f, err := strconv.ParseFloat(def.Value, 64)
			if err == nil {
				if vars.FloatVariables == nil {
					vars.FloatVariables = make(core.VariableMap[float64])
				}
				vars.FloatVariables[key] = core.Variable[float64]{
					Key:   key,
					Value: f,
					Type:  string(*def.Type),
				}
			}

		case features.VariableValueTypeJson:
			if vars.JsonVariables == nil {
				vars.JsonVariables = make(core.VariableMap[any])
			}
			vars.JsonVariables[key] = core.Variable[any]{
				Key:   key,
				Value: def.Value,
				Type:  string(*def.Type),
			}

		case features.VariableValueTypeString:
			if vars.StringVariables == nil {
				vars.StringVariables = make(core.VariableMap[string])
			}
			vars.StringVariables[key] = core.Variable[string]{
				Key:   key,
				Value: def.Value,
				Type:  string(*def.Type),
			}
		}
	}

	return vars
}

func parseVariableDefinitions(defs *map[string]features.VariableDefinition) core.Variables {
	vars := core.Variables{}
	if defs == nil {
		return vars
	}

	for key, def := range *defs {
		switch def.Type {
		case features.VariableDefinitionTypeBoolean:
			b, err := strconv.ParseBool(def.DefaultValue)
			if err == nil {
				if vars.BoolVariables == nil {
					vars.BoolVariables = make(core.VariableMap[bool])
				}
				vars.BoolVariables[key] = core.Variable[bool]{
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
					vars.IntVariables = make(core.VariableMap[int])
				}
				vars.IntVariables[key] = core.Variable[int]{
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
					vars.FloatVariables = make(core.VariableMap[float64])
				}
				vars.FloatVariables[key] = core.Variable[float64]{
					Key:         key,
					Value:       f,
					Type:        string(def.Type),
					Description: optionalString(def.Description),
				}
			}

		case features.VariableDefinitionTypeJson:
			if vars.JsonVariables == nil {
				vars.JsonVariables = make(core.VariableMap[any])
			}
			vars.JsonVariables[key] = core.Variable[any]{
				Key:         key,
				Value:       def.DefaultValue,
				Type:        string(def.Type),
				Description: optionalString(def.Description),
			}

		case features.VariableDefinitionTypeString:
			if vars.StringVariables == nil {
				vars.StringVariables = make(core.VariableMap[string])
			}
			vars.StringVariables[key] = core.Variable[string]{
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
	response, err := f.client.ListVariableDefinitionsWithResponse(ctx, projectId, flagKey, &features.ListVariableDefinitionsParams{
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
			pageResp, err := f.client.ListVariableDefinitionsWithResponse(ctx, projectId, flagKey, &features.ListVariableDefinitionsParams{
				PageToken:  &token,
				PerPage:    new(largePageSize),
				PageWindow: new(1),
			})
			if err != nil {
				log.Warn("failed to fetch next page of variable definitions", "flag", flagKey, "error", err.Error())
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
	response, err := f.client.ListVariationsWithResponse(ctx, projectId, flagKey, &features.ListVariationsParams{
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
			pageResp, err := f.client.ListVariationsWithResponse(ctx, projectId, flagKey, &features.ListVariationsParams{
				PageToken:  &token,
				PerPage:    new(largePageSize),
				PageWindow: new(1),
			})
			if err != nil {
				log.Warn("failed to fetch next page of variations", "flag", flagKey, "error", err.Error())
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
func (f *flagDataMapper) enrichFlag(ctx context.Context, flag *features.Flag) (*core.FeatureFlagDefinition, error) {
	flag.VariableDefinitions = new(getAllDefinitions(ctx, flag, f))
	// Fetch all variations
	allVariations, err := f.fetchAllVariations(ctx, flag.Key)
	if err != nil {
		log.Warn("failed to fetch all variations", "flag", flag.Key, "error", err.Error())
	}

	result, err := toFeatureFlag(*flag)
	if err != nil {
		return nil, err
	}
	overrides := make([]core.Override, 0)
	for _, variation := range allVariations {
		overrides = append(overrides, toOverride(variation))
	}
	result.Overrides = overrides

	if result.Meta == nil {
		result.Meta = make(map[string]any)
	}
	result.Meta["enriched"] = true

	return &result, nil
}

func getAllDefinitions(ctx context.Context, flag *features.Flag, f *flagDataMapper) map[string]features.VariableDefinition {
	definitions := make(map[string]features.VariableDefinition)
	// Fetch all variable definitions if the flag has more than 5
	if flag.VariableDefinitions != nil {
		definitions = *flag.VariableDefinitions
		if len(definitions) > 5 {
			allDefs, err := f.fetchAllVariableDefinitions(ctx, flag.Key)
			if err != nil {
				log.Warn("failed to fetch all variable definitions", "flag", flag.Key, "error", err.Error())
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

func toVariableDefinitionMap(vars core.Variables) *map[string]features.VariableDefinition {
	defs := vars.Definitions()
	if len(defs) == 0 {
		return nil
	}

	result := make(map[string]features.VariableDefinition, len(defs))
	for key, def := range defs {
		item := features.VariableDefinition{
			Key:          key,
			Type:         features.VariableDefinitionType(def.Type),
			DefaultValue: def.DefaultValue,
		}
		if def.Description != "" {
			item.Description = new(def.Description)
		}
		result[key] = item
	}
	return &result
}

func (f *flagDataMapper) AddVariables(ctx context.Context, flagKey string, vars core.Variables) error {
	err := f.validate()
	if err != nil {
		return err
	}
	projectID, err := f.getIdForService()
	if err != nil {
		return err
	}

	defs := vars.Definitions()
	if len(defs) == 0 {
		return nil
	}

	keys := make([]string, 0, len(defs))
	for key := range defs {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		def := defs[key]
		body := features.VariableDefinition{
			Key:          def.Key,
			Type:         features.VariableDefinitionType(def.Type),
			DefaultValue: def.DefaultValue,
		}
		if def.Description != "" {
			body.Description = new(def.Description)
		}

		response, err := f.client.CreateVariableDefinitionWithResponse(ctx, projectID, flagKey, body)
		if err != nil {
			return fmt.Errorf("failed to create variable definition %s on flag %s: %w", key, flagKey, err)
		}
		if response.StatusCode() != 200 {
			return fmt.Errorf("non-200 status code creating variable definition %s on flag %s: %d", key, flagKey, response.StatusCode())
		}
	}

	return nil
}

func (f *flagDataMapper) Delete(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("delete not supported for flags")
}

func (f *flagDataMapper) Create(ctx context.Context, value core.FeatureFlagDefinition) (*core.FeatureFlagDefinition, error) {
	err := f.validate()
	if err != nil {
		return nil, err
	}

	id, err := f.getIdForService()
	if err != nil {
		return nil, err
	}

	toPtr := func(s string) *string {
		if s == "" {
			return nil
		}
		return new(s)
	}

	varDefs := make(map[string]features.VariableDefinition)

	for k, v := range value.DefaultVariables.BoolVariables {
		varDefs[k] = features.VariableDefinition{
			Key:          v.Key,
			Type:         features.VariableDefinitionTypeBoolean,
			DefaultValue: strconv.FormatBool(v.Value),
			Description:  toPtr(v.Description),
		}
	}
	for k, v := range value.DefaultVariables.IntVariables {
		varDefs[k] = features.VariableDefinition{
			Key:          v.Key,
			Type:         features.VariableDefinitionTypeInteger,
			DefaultValue: strconv.Itoa(v.Value),
			Description:  toPtr(v.Description),
		}
	}
	for k, v := range value.DefaultVariables.FloatVariables {
		varDefs[k] = features.VariableDefinition{
			Key:          v.Key,
			Type:         features.VariableDefinitionTypeDouble,
			DefaultValue: strconv.FormatFloat(v.Value, 'f', -1, 64),
			Description:  toPtr(v.Description),
		}
	}
	for k, v := range value.DefaultVariables.StringVariables {
		varDefs[k] = features.VariableDefinition{
			Key:          v.Key,
			Type:         features.VariableDefinitionTypeString,
			DefaultValue: v.Value,
			Description:  toPtr(v.Description),
		}
	}
	for k, v := range value.DefaultVariables.JsonVariables {
		var valStr string
		switch typed := v.Value.(type) {
		case string:
			valStr = typed
		default:
			bytes, err := json.Marshal(v.Value)
			if err == nil {
				valStr = string(bytes)
			}
		}
		varDefs[k] = features.VariableDefinition{
			Key:          v.Key,
			Type:         features.VariableDefinitionTypeJson,
			DefaultValue: valStr,
			Description:  toPtr(v.Description),
		}
	}

	body := features.CreateFlagJSONRequestBody{
		Key:         value.Key,
		Name:        toPtr(value.Name),
		Description: toPtr(value.Description),
	}
	if len(varDefs) > 0 {
		body.VariableDefinitions = &varDefs
	}

	response, err := f.client.CreateFlagWithResponse(ctx, id, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create flag. %w", err)
	}
	if response.StatusCode() != 201 {
		detail := ""
		if response.ApplicationproblemJSON400 != nil {
			detail = response.ApplicationproblemJSON400.Detail
		}
		return nil, fmt.Errorf("non-201 status code: %d, detail: %s", response.StatusCode(), detail)
	}

	if response.JSON201 == nil {
		return nil, fmt.Errorf("missing created flag response")
	}

	// Create variations
	for _, override := range value.Overrides {
		varValues := make(map[string]features.VariableValue)
		for k, v := range override.Variables.BoolVariables {
			t := features.VariableValueTypeBoolean
			varValues[k] = features.VariableValue{
				Key:   toPtr(v.Key),
				Type:  &t,
				Value: strconv.FormatBool(v.Value),
			}
		}
		for k, v := range override.Variables.IntVariables {
			t := features.VariableValueTypeInteger
			varValues[k] = features.VariableValue{
				Key:   toPtr(v.Key),
				Type:  &t,
				Value: strconv.Itoa(v.Value),
			}
		}
		for k, v := range override.Variables.FloatVariables {
			t := features.VariableValueTypeDouble
			varValues[k] = features.VariableValue{
				Key:   toPtr(v.Key),
				Type:  &t,
				Value: strconv.FormatFloat(v.Value, 'f', -1, 64),
			}
		}
		for k, v := range override.Variables.StringVariables {
			t := features.VariableValueTypeString
			varValues[k] = features.VariableValue{
				Key:   toPtr(v.Key),
				Type:  &t,
				Value: v.Value,
			}
		}
		for k, v := range override.Variables.JsonVariables {
			var valStr string
			switch typed := v.Value.(type) {
			case string:
				valStr = typed
			default:
				bytes, err := json.Marshal(v.Value)
				if err == nil {
					valStr = string(bytes)
				}
			}
			t := features.VariableValueTypeJson
			varValues[k] = features.VariableValue{
				Key:   toPtr(v.Key),
				Type:  &t,
				Value: valStr,
			}
		}

		varBody := features.CreateVariationJSONRequestBody{
			Key:         override.Key,
			Name:        override.Name,
			Description: toPtr(override.Description),
		}
		if len(varValues) > 0 {
			varBody.Variables = &varValues
		}

		varResp, err := f.client.CreateVariationWithResponse(ctx, id, value.Key, varBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create variation %s: %w", override.Key, err)
		}
		if varResp.StatusCode() != 201 {
			detail := ""
			if varResp.ApplicationproblemJSON400 != nil {
				detail = varResp.ApplicationproblemJSON400.Detail
			}
			return nil, fmt.Errorf("failed to create variation %s (status %d): %s", override.Key, varResp.StatusCode(), detail)
		}
	}

	// Disable ruleset and set default variation to "off" (if it exists) for all environments
	// to ensure the flag starts inactive / in draft mode and defaults to the off variant.
	if response.JSON201.Environments != nil {
		for envKey := range *response.JSON201.Environments {
			_, disableErr := f.client.DisableRulesetWithResponse(ctx, id, value.Key, envKey)
			if disableErr != nil {
				log.Warn("Failed to explicitly disable ruleset for environment", "env", envKey, "error", disableErr.Error())
			}

			var hasOffVariation bool
			for _, o := range value.Overrides {
				if o.Key == "off" {
					hasOffVariation = true
					break
				}
			}
			if hasOffVariation {
				var val features.PatchRequestBody_Value
				err = val.FromPatchRequestBodyValue3("off")
				if err == nil {
					patchBody := []features.PatchRequestBody{
						{
							Op:    features.Replace,
							Path:  "/default_variation_key",
							Value: &val,
						},
					}
					_, patchErr := f.client.UpdateRulesetWithApplicationJSONPatchPlusJSONBodyWithResponse(ctx, id, value.Key, envKey, patchBody)
					if patchErr != nil {
						log.Warn("Failed to set default variation to off for environment", "env", envKey, "error", patchErr.Error())
					}
				}
			}
		}
	}

	enriched, err := f.enrichFlag(ctx, response.JSON201)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich newly created flag: %w", err)
	}

	return enriched, nil
}

func (f *flagDataMapper) Update(_ context.Context, _ func(value *core.FeatureFlagDefinition) error) (*core.FeatureFlagDefinition, error) {
	return nil, fmt.Errorf("update not supported for flags")
}
