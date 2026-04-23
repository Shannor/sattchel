package optimizely

import (
	"github.com/spf13/viper"
)

type Repository struct {
	v *viper.Viper
}

func NewConfigRepo(v *viper.Viper) *Repository {
	return &Repository{v: v}
}

func (r *Repository) GetConfig() (*Configuration, error) {
	var cfg Configuration
	err := r.v.UnmarshalKey("optimizely", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetConfig updates just the Optimizely part and saves to disk
func (r *Repository) SetConfig(config Configuration) error {
	r.v.Set("optimizely", config)
	return r.v.WriteConfig()
}
