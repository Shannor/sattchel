package optimizely

import (
	"context"
	"fmt"
	"net/http"

	"sattchel/internal/domain"
)

// Configuration holds settings for API access and associated projects.
type Configuration struct {
	APIKey         string                          `mapstructure:"apiKey" json:"apiKey" yaml:"apiKey"`
	Projects       []domain.Project                `mapstructure:"projects" json:"projects" yaml:"projects"`
	EnvironmentMap map[string][]domain.Environment `json:"environmentMap" mapstructure:"environmentMap" yaml:"environmentMap"`
}

// WithToken returns a RequestEditorFn that injects the auth header.
func WithToken(token string) func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
	}
}
