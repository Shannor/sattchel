package configs

type Config struct {
	Optimizely OptimizelyConfig `mapstructure:"optimizely" json:"optimizely"`
	Contentful ContentfulConfig `mapstructure:"contentful" json:"contentful"`
}
type OptimizelyConfig struct {
	APIKey   string   `mapstructure:"apiKey" json:"apiKey"`
	Projects []string `mapstructure:"projects" json:"projects"`
}

type ContentfulConfig struct {
	APIKey string `mapstructure:"apiKey" json:"apiKey"`
}
