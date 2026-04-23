package optimizely

import (
	"errors"
	"test-cli/internal/repository"
)

type Service interface {
	GetConfig() (*Configuration, error)
	SetConfig(config Configuration) error
}

type service struct {
	repo repository.Configuration[Configuration]
}

func (o service) GetConfig() (*Configuration, error) {
	return o.repo.GetConfig()
}

func (o service) SetConfig(config Configuration) error {
	if config.APIKey == "" {
		return errors.New("API key cannot be empty")
	}
	return o.repo.SetConfig(config)
}

func NewOptimizelyService(repo repository.Configuration[Configuration]) Service {
	return &service{
		repo: repo,
	}
}
