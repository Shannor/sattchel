package config

import (
	"context"

	"github.com/spf13/viper"
)

type OptimizelyRepository struct {
	v *viper.Viper
}

type optKey struct{}

func (r *OptimizelyRepository) GetConfig() (*OptimizelyConfig, error) {
	var cfg OptimizelyConfig
	err := r.v.UnmarshalKey("optimizely", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetConfig updates just the Optimizely part and saves to disk
func (r *OptimizelyRepository) SetConfig(config OptimizelyConfig) error {
	r.v.Set("optimizely", config)
	return r.v.WriteConfig()
}

func (r *OptimizelyRepository) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, optKey{}, r)
}

func (r *OptimizelyRepository) FromContext(ctx context.Context) Service {
	return ctx.Value(optKey{}).(Service)
}
