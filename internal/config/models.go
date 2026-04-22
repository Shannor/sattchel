package config

type OptimizelyConfig struct {
	APIKey   string   `mapstructure:"apiKey" json:"apiKey"`
	Projects []string `mapstructure:"projects" json:"projects"`
}

type ContentfulConfig struct {
	APIKey string `mapstructure:"apiKey" json:"apiKey"`
}

type Repository[T any] interface {
	SetConfig(config T) error
	GetConfig() (*T, error)
}
