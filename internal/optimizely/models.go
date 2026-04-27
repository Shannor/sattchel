package optimizely

// Configuration holds settings for API access and associated projects.
type Configuration struct {
	APIKey   string    `mapstructure:"apiKey" json:"apiKey" yaml:"apiKey"`
	Projects []Project `mapstructure:"projects" json:"projects" yaml:"projects"`
}

type Project struct {
	ID       string `json:"id" mapstructure:"id" yaml:"id"`
	Name     string `json:"name" mapstructure:"name" yaml:"name"`
	IsActive bool   `json:"isActive" mapstructure:"isActive" yaml:"isActive"`
}
