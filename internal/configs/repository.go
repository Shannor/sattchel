package configs

import (
	"fmt"

	"github.com/spf13/viper"
)

type ConfigRepo interface {
	SetConfig(config Config) error
	GetConfig() (*Config, error)
	Init() error
}

type configRepo struct {
}

func NewConfigRepo() ConfigRepo {
	return &configRepo{}
}

// SetConfig updates the entire config file
func (r *configRepo) SetConfig(config Config) error {
	viper.Set("optimizely", config.Optimizely)
	viper.Set("contentful", config.Contentful)
	return nil
}

// GetConfig returns the Config representation of the config file
func (r *configRepo) GetConfig() (*Config, error) {
	c := Config{}
	err := viper.Unmarshal(&c)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return c, nil
}

func (r *configRepo) Init() error {
	return nil
}
