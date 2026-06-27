package optimizely

import (
	"context"
	"fmt"
	"sattchel/internal/domain"

	"github.com/spf13/viper"
)

type configDataMapper struct {
	v *viper.Viper
}

// ConfigDataMapper is the data mapper interface for Optimizely configuration.
type ConfigDataMapper domain.DataMapper[Configuration]

func NewConfigDM(v *viper.Viper) ConfigDataMapper {
	return &configDataMapper{v: v}
}

func (c *configDataMapper) Get(ctx context.Context, ID string) (*Configuration, error) {
	var cfg Configuration
	err := c.v.UnmarshalKey("optimizely", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *configDataMapper) GetAll(ctx context.Context) ([]Configuration, error) {
	cfg, err := c.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("missing configuration")
	}
	return []Configuration{*cfg}, nil
}

func (c *configDataMapper) Create(ctx context.Context, value Configuration) (*Configuration, error) {
	c.v.Set("optimizely", value)
	if err := c.v.WriteConfig(); err != nil {
		return nil, err
	}
	return &value, nil
}

func (c *configDataMapper) Update(ctx context.Context, updater func(value *Configuration) error) (*Configuration, error) {
	cfg, err := c.Get(ctx, "")
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

func (c *configDataMapper) Delete(ctx context.Context, ID string) (string, error) {
	c.v.Set("optimizely", nil)
	return "", c.v.WriteConfig()
}
