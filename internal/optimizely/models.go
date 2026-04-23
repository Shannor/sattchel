package optimizely

// Configuration holds settings for API access and associated projects.
type Configuration struct {
	APIKey   string   `mapstructure:"apiKey" json:"apiKey"`
	Projects []string `mapstructure:"projects" json:"projects"`
	Count    int      `mapstructure:"count" json:"count"`
}
