package driven

import (
	"context"
	"fmt"
	"sattchel/internal/optimizely/core"

	"github.com/spf13/viper"
)

type configDataMapper struct {
	v *viper.Viper
}

func NewConfigDM(v *viper.Viper) core.ConfigRepository {
	return &configDataMapper{v: v}
}

func (c *configDataMapper) Get(ctx context.Context, ID string) (*core.Configuration, error) {
	var cfg core.Configuration
	err := c.v.UnmarshalKey("optimizely", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *configDataMapper) GetAll(ctx context.Context) ([]core.Configuration, error) {
	cfg, err := c.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("missing configuration")
	}
	return []core.Configuration{*cfg}, nil
}

func (c *configDataMapper) Create(ctx context.Context, value core.Configuration) (*core.Configuration, error) {
	c.v.Set("optimizely", value)
	if err := c.v.WriteConfig(); err != nil {
		return nil, err
	}
	return &value, nil
}

func (c *configDataMapper) Update(ctx context.Context, updater func(value *core.Configuration) error) (*core.Configuration, error) {
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
