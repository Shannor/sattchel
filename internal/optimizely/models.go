package optimizely

import (
	"test-cli/internal/models"
)

// Configuration holds settings for API access and associated projects.
type Configuration struct {
	APIKey   string           `mapstructure:"apiKey" json:"apiKey" yaml:"apiKey"`
	Projects []models.Project `mapstructure:"projects" json:"projects" yaml:"projects"`
}
