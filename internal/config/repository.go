package config

import (
	"context"

	"github.com/spf13/viper"
)

type ContentfulRepository struct {
	v *viper.Viper
}

type contentfulKey struct{}

func (r *ContentfulRepository) GetConfig() (*ContentfulConfig, error) {
	var cfg ContentfulConfig
	err := r.v.UnmarshalKey("contenful", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetConfig updates just the Optimizely part and saves to disk
func (r *ContentfulRepository) SetConfig(config ContentfulConfig) error {
	r.v.Set("contentful", config)

	return r.v.WriteConfig()
}

func (r *ContentfulRepository) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contentfulKey{}, r)
}

func (r *ContentfulRepository) FromContext(ctx context.Context) Service {
	return ctx.Value(contentfulKey{}).(Service)
}
