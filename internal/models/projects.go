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

type Environment struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	Key       string         `json:"key"`
	Name      string         `json:"name"`
	IsActive  bool           `json:"isActive"`
	Meta      map[string]any `json:"meta"`
}
