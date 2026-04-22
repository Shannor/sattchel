package repository

type FeatureFlagRepository interface {
	GetFlags() ([]FeatureFlag, error)
	GetFlag() (*FeatureFlag, error)
}

// FeatureFlag represents a feature flag with various variable types
// TODO: May use on struct for all the variable types
type FeatureFlag struct {
	Name      string    `json:"name"`
	Variables Variables `json:"variables"`
}

type Variables struct {
	FloatVariables  VariableMap[float64] `json:"floatVariables"`
	IntVariables    VariableMap[int]     `json:"intVariables"`
	StringVariables VariableMap[string]  `json:"stringVariables"`
	JsonVariables   VariableMap[any]     `json:"jsonVariables"`
}
type VariableMap[T any] map[string]Variable[T]
type Variable[T any] struct {
	Value T
	Type  string
}
