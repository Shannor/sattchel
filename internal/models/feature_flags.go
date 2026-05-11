package models

import "time"

// FeatureFlag represents a feature flag with various variable types
type FeatureFlag struct {
	// ID system name for the feature flag
	ID string `json:"id"`
	// Name the user facing label of the feature flag
	Name       string `json:"name"`
	IsArchived bool   `json:"isArchived"`
	// DefaultVariables additional configuration for features flags. Only used when enabled.
	DefaultVariables Variables `json:"defaultVariables"`
	// Overrides particular overrides on an instance of a flag.
	Overrides   []FlagOverride `json:"overrides"`
	Description string         `json:"description"`
	// Environments shows from the list of environments which this flag is active for
	Environments []Environment  `json:"environments"`
	Meta         map[string]any `json:"meta"`
	CreatedBy    *string        `json:"createdBy"`
	CreatedAt    *time.Time     `json:"createdAt"`
}

type FlagOverride struct {
	ID             string    `json:"id"`
	FlagID         string    `json:"flagId"`
	EnvironmentIDs []string  `json:"environmentIDs"`
	Variables      Variables `json:"variables"`
	Enabled        bool      `json:"enabled"`
	Archived       bool      `json:"archived"`
}

type Variables struct {
	FloatVariables  VariableMap[float64] `json:"floatVariables"`
	IntVariables    VariableMap[int]     `json:"intVariables"`
	StringVariables VariableMap[string]  `json:"stringVariables"`
	JsonVariables   VariableMap[any]     `json:"jsonVariables"`
	BoolVariables   VariableMap[bool]    `json:"boolVariables"`
}
type VariableMap[T any] map[string]Variable[T]
type Variable[T any] struct {
	Value T
	Type  string
}

type Environment struct {
	ID      string         `json:"id"`
	Key     string         `json:"key"`
	Name    string         `json:"name"`
	Enabled bool           `json:"enabled"`
	Meta    map[string]any `json:"meta"`
}
