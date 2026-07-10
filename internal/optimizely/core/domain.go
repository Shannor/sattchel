package core

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Project represents any project in the system. Similar to a tenant.
// Everything will be below this.
type Project struct {
	ID       string `json:"id" mapstructure:"id" yaml:"id"`
	Name     string `json:"name" mapstructure:"name" yaml:"name"`
	IsActive bool   `json:"isActive" mapstructure:"isActive" yaml:"isActive"`
	Label    string `json:"label" mapstructure:"label" yaml:"label"`
	Source   string `json:"source" mapstructure:"source" yaml:"source"`
}

// FlagComparison represents a comparison status of a feature flag across projects.
type FlagComparison struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	ExistsIn  []Project `json:"existsIn"`
	MissingIn []Project `json:"missingIn"`
}

// Environment Top level class under Project. Will generally exist in an array.
// Will have many to many mappings with other Models generally.
type Environment struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	Key       string         `json:"key"`
	Name      string         `json:"name"`
	Archived  bool           `json:"archived"`
	Meta      map[string]any `json:"meta"`
}

// Target is a mapping with information between Environment and FeatureFlagDefinition.
// It has relationship data that will inform statuses of a FeatureFlagInstance.
// It will be nested within the FeatureFlagDefinition and private.
type Target struct {
	EnvironmentID string `json:"environmentId"`
	OverrideID    string `json:"overrideId,omitempty"`
	IsEnabled     bool   `json:"isEnabled"`
}

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
	// Overrides a collection of overriding variables that help build the instance of a Feature Flag.
	Overrides []Override `json:"overrides,omitempty"`
	// Targets a collection that provides the mapping between FeatureFlagDefinition, Environment, and Override
	Targets []Target `json:"targets,omitempty"`
}

func (f *FeatureFlagDefinition) AllInstances() ([]FeatureFlagInstance, error) {
	result := make([]FeatureFlagInstance, len(f.Targets))
	for _, target := range f.Targets {
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

	if f.Overrides == nil || len(f.Overrides) == 0 {
		return &result, nil
	}

	idx := slices.IndexFunc(f.Targets, func(target Target) bool {
		return envID == target.EnvironmentID
	})
	if idx == -1 {
		return &result, nil
	}

	t := f.Targets[idx]
	result.Enabled = t.IsEnabled
	if t.OverrideID == "" {
		return &result, nil
	}

	idx = slices.IndexFunc(f.Overrides, func(override Override) bool {
		return t.OverrideID == override.Key || t.OverrideID == override.ID
	})
	if idx == -1 {
		return &result, nil
	}
	o := f.Overrides[idx]
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

// UnmarshalJSON reconstructs the typed Variable maps from a flat JSON object.
func (v *Variables) UnmarshalJSON(data []byte) error {
	var flat map[string]any
	if err := json.Unmarshal(data, &flat); err != nil {
		return err
	}

	v.BoolVariables = make(VariableMap[bool])
	v.IntVariables = make(VariableMap[int])
	v.FloatVariables = make(VariableMap[float64])
	v.StringVariables = make(VariableMap[string])
	v.JsonVariables = make(VariableMap[any])

	for key, val := range flat {
		switch valT := val.(type) {
		case bool:
			v.BoolVariables[key] = Variable[bool]{
				Key:   key,
				Value: valT,
				Type:  "boolean",
			}
		case string:
			trimmed := strings.TrimSpace(valT)
			if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
				(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
				var parsed any
				if err := json.Unmarshal([]byte(valT), &parsed); err == nil {
					v.JsonVariables[key] = Variable[any]{
						Key:   key,
						Value: valT,
						Type:  "json",
					}
					continue
				}
			}
			v.StringVariables[key] = Variable[string]{
				Key:   key,
				Value: valT,
				Type:  "string",
			}
		case float64:
			if valT == float64(int(valT)) {
				v.IntVariables[key] = Variable[int]{
					Key:   key,
					Value: int(valT),
					Type:  "integer",
				}
			} else {
				v.FloatVariables[key] = Variable[float64]{
					Key:   key,
					Value: valT,
					Type:  "double",
				}
			}
		case map[string]any, []any:
			v.JsonVariables[key] = Variable[any]{
				Key:   key,
				Value: valT,
				Type:  "json",
			}
		default:
			v.JsonVariables[key] = Variable[any]{
				Key:   key,
				Value: valT,
				Type:  "json",
			}
		}
	}
	return nil
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
