package models

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

// FeatureFlagDefinition represents a feature flag with various variable types
// A feature flag must exist in at least one environment to be real.
type FeatureFlagDefinition struct {
	// ID Unique key for the definition across all projects
	ID string `json:"id"`
	// Key Unique key that's the same across projects but unique within the project itself
	Key string `json:"key"`
	// Name the user-facing label
	Name     string `json:"name"`
	Archived bool   `json:"archived"`
	// DefaultVariables additional configuration for features flags. Only used when enabled.
	DefaultVariables Variables      `json:"defaultVariables"`
	Description      string         `json:"description"`
	Meta             map[string]any `json:"meta"`
	CreatedBy        *string        `json:"createdBy"`
	CreatedAt        *time.Time     `json:"createdAt"`
	// overrides a collection of overriding variables that help build the instance of a Feature Flag.
	// Private since the creation of the Feature Flag should be handled by models.
	overrides []Override
	// targets a collection that provides the mapping between FeatureFlagDefinition, Environment, and Override
	targets []Target
}

func (f *FeatureFlagDefinition) SetOverrides(o []Override) {
	f.overrides = o
}

func (f *FeatureFlagDefinition) SetTargets(target []Target) {
	f.targets = target
}

func (f *FeatureFlagDefinition) AllInstances() ([]FeatureFlagInstance, error) {
	result := make([]FeatureFlagInstance, len(f.targets))
	for _, target := range f.targets {
		i, err := f.ByEnvID(target.EnvironmentID)
		if err != nil {
			continue
		}
		result = append(result, *i)
	}
	return result, nil
}

func (f *FeatureFlagDefinition) ByEnvID(envID string) (*FeatureFlagInstance, error) {
	result := FeatureFlagInstance{
		ID:            f.ID,
		Name:          f.Name,
		Description:   f.Description,
		Variables:     f.DefaultVariables,
		EnvironmentID: envID,
		Archived:      f.Archived,
	}

	if f.overrides == nil || len(f.overrides) == 0 {
		return &result, nil
	}

	idx := slices.IndexFunc(f.targets, func(target Target) bool {
		return envID == target.EnvironmentID
	})
	if idx == -1 {
		return &result, nil
	}

	t := f.targets[idx]
	result.Enabled = t.IsEnabled
	if t.OverrideID == "" {
		return &result, nil
	}

	idx = slices.IndexFunc(f.overrides, func(override Override) bool {
		return t.OverrideID == override.Key || t.OverrideID == override.ID
	})
	if idx == -1 {
		return &result, nil
	}
	o := f.overrides[idx]
	result.Variables.Merge(o.Variables)
	return &result, nil
}

type FeatureFlagInstance struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Variables     Variables `json:"variables"`
	EnvironmentID string    `json:"environmentId"`
	Description   string    `json:"description"`
	Enabled       bool      `json:"enabled,omitempty"`
	Archived      bool      `json:"archived,omitempty"`
}

type Override struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Variables   Variables `json:"variables"`
}

type Variables struct {
	FloatVariables  VariableMap[float64] `json:"floatVariables"`
	IntVariables    VariableMap[int]     `json:"intVariables"`
	StringVariables VariableMap[string]  `json:"stringVariables"`
	JsonVariables   VariableMap[any]     `json:"jsonVariables"`
	BoolVariables   VariableMap[bool]    `json:"boolVariables"`
}

func (v Variables) Merge(other Variables) {
	for key, val := range other.BoolVariables {
		v.BoolVariables[key] = val
	}
	for key, val := range other.IntVariables {
		v.IntVariables[key] = val
	}
	for key, val := range other.StringVariables {
		v.StringVariables[key] = val
	}
	for key, val := range other.JsonVariables {
		v.JsonVariables[key] = val
	}
	for key, val := range other.FloatVariables {
		v.FloatVariables[key] = val
	}
}

// MarshalJSON flattens the Variables into a single JSON object where keys are
// variable names and values are their raw values (not wrapped in Variable[T]).
func (v Variables) MarshalJSON() ([]byte, error) {
	flat := make(map[string]any)

	for key, val := range v.FloatVariables {
		flat[key] = val.Value
	}
	for key, val := range v.IntVariables {
		flat[key] = val.Value
	}
	for key, val := range v.StringVariables {
		flat[key] = val.Value
	}
	for key, val := range v.JsonVariables {
		flat[key] = val.Value
	}
	for key, val := range v.BoolVariables {
		flat[key] = val.Value
	}

	return json.Marshal(flat)
}

// String returns the JSON representation of Variables.
func (v Variables) String() string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("Variables{marshal error: %v}", err)
	}
	return string(b)
}

type VariableMap[T any] map[string]Variable[T]
type Variable[T any] struct {
	Key         string `json:"key"`
	Value       T      `json:"value"`
	Type        string `json:"type"`
	Description string `json:"description"`
}
