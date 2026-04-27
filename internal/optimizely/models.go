package optimizely

// Configuration holds settings for API access and associated projects.
type Configuration struct {
	APIKey         string    `mapstructure:"apiKey" json:"apiKey"`
	DefaultAccount string    `mapstructure:"defaultAccount" json:"defaultAccount"`
	Projects       []Project `mapstructure:"projects" json:"projects"`
	Count          int       `mapstructure:"count" json:"count"`
}

type Project struct {
	ID       string `json:"id" mapstructure:"id"`
	Name     string `json:"name" mapstructure:"name"`
	IsActive bool   `json:"isActive" mapstructure:"isActive"`
}
