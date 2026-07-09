package driving

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	v *viper.Viper
}

func LoadConfig(v *viper.Viper) (*Config, error) {
	return &Config{v: v}, nil

}

type Configuration struct {
	CurrentProjectID string `mapstructure:"currentProjectId" json:"currentProjectId" yaml:"currentProjectId"`
	CurrentGoalID    string `mapstructure:"currentGoalId" json:"currentGoalId" yaml:"currentGoalId"`
}

func (c *Config) Get() (*Configuration, error) {
	var cfg Configuration
	err := c.v.UnmarshalKey("tracker", &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracker config: %w", err)
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
	c.v.Set("tracker", cfg)
	if err := c.v.WriteConfig(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) CurrentProjectID() string {
	cfg, err := c.Get()
	if err != nil {
		return ""
	}
	return cfg.CurrentProjectID
}

func (c *Config) SetCurrentProjectID(id string) error {
	_, err := c.Update(func(cfg *Configuration) error {
		cfg.CurrentProjectID = id
		return nil
	})
	return err
}

func (c *Config) CurrentGoalID() string {
	cfg, err := c.Get()
	if err != nil {
		return ""
	}
	return cfg.CurrentGoalID
}

func (c *Config) SetCurrentGoalID(id string) error {
	_, err := c.Update(func(cfg *Configuration) error {
		cfg.CurrentGoalID = id
		return nil
	})
	return err
}
