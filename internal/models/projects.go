package models

// Project represents any project in the system. Similar to a tenant.
// Everything will be below this.
type Project struct {
	ID       string `json:"id" mapstructure:"id" yaml:"id"`
	Name     string `json:"name" mapstructure:"name" yaml:"name"`
	IsActive bool   `json:"isActive" mapstructure:"isActive" yaml:"isActive"`
	Label    string `json:"label" mapstructure:"label" yaml:"label"`
	Source   string `json:"source" mapstructure:"source" yaml:"source"`
}

// Environment Top level class under Project. Will generally exist in an array.
// Will have manu to many mappings with other Models generally.
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
