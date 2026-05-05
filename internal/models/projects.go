package models

// Project represents any project in the system. Can be for Optimizely or Contentful
type Project struct {
	ID       string `json:"id" mapstructure:"id" yaml:"id"`
	Name     string `json:"name" mapstructure:"name" yaml:"name"`
	IsActive bool   `json:"isActive" mapstructure:"isActive" yaml:"isActive"`
	Label    string `json:"label" mapstructure:"label" yaml:"label"`
}
