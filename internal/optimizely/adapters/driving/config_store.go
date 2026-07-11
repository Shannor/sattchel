package driving

import (
	"fmt"
	"sattchel/internal/optimizely/core"

	"github.com/spf13/viper"
)

type Configuration struct {
	APIKey          string                        `mapstructure:"apiKey" json:"apiKey" yaml:"apiKey"`
	Projects        []core.Project                `mapstructure:"projects" json:"projects" yaml:"projects"`
	EnvironmentMap  map[string][]core.Environment `json:"environmentMap" mapstructure:"environmentMap" yaml:"environmentMap"`
	CacheTTLMinutes int64                         `mapstructure:"cacheTTLMinutes" json:"cacheTTLMinutes" yaml:"cacheTTLMinutes"`
}

type Config struct {
	v *viper.Viper
}

func NewConfig(v *viper.Viper) *Config {
	return &Config{v: v}
}

func (c *Config) Get() (*Configuration, error) {
	var cfg Configuration
	err := c.v.UnmarshalKey("optimizely", &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Update(updater func(value *Configuration) error) (*Configuration, error) {
	cfg, err := c.Get()
	if err != nil {
		return nil, err
	}
	if err := updater(cfg); err != nil {
		return nil, err
	}
	c.v.Set("optimizely", *cfg)
	if err := c.v.WriteConfig(); err != nil {
		return nil, err
	}
	return cfg, nil
}
