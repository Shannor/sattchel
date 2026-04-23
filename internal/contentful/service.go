package contentful

import "test-cli/internal/repository"

type Service interface {
	GetConfig() (*Configuration, error)
	SetConfig(config Configuration) error
}

type contentfulService struct {
	repo repository.Configuration[Configuration]
}

func (s contentfulService) GetConfig() (*Configuration, error) {
	//TODO implement me
	panic("implement me")
}

func (s contentfulService) SetConfig(config Configuration) error {
	//TODO implement me
	panic("implement me")
}

func NewConfigurationService(repo repository.Configuration[Configuration]) Service {
	return &contentfulService{
		repo: repo,
	}
}
