package models

// FeatureFlag represents a feature flag with various variable types
type FeatureFlag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Variables Variables `json:"variables"`
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
