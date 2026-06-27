package contentful

import (
	"sattchel/internal/models"

	"github.com/spf13/viper"
)

type Repository struct {
	v *viper.Viper
}

func NewConfigRepo(v *viper.Viper) models.Configuration[Configuration] {
	return &Repository{v: v}
}

func (r *Repository) GetConfig() (*Configuration, error) {
	var cfg Configuration
	err := r.v.UnmarshalKey("contenful", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetConfig updates just the Optimizely part and saves to disk
func (r *Repository) SetConfig(config Configuration) error {
	r.v.Set("contentful", config)

	return r.v.WriteConfig()
}
