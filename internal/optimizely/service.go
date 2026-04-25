package optimizely

import (
	"errors"
	"test-cli/internal/repository"
)

type Service interface {
	GetAccounts() ([]string, error)
	GetProjects(accountIds []string) ([]string, error)
	GetConfig() (*Configuration, error)
	SetConfig(config Configuration) error
}

type service struct {
	repo repository.Configuration[Configuration]
}

func (o service) GetAccounts() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (o service) GetProjects(accountIds []string) ([]string, error) {
	//TODO implement me
	panic("implement me")
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
