package contentful

import "sattchel/internal/models"

type Service interface {
	GetConfig() (*Configuration, error)
	SetConfig(config Configuration) error
}

type contentfulService struct {
	repo models.Configuration[Configuration]
}

func (s contentfulService) GetConfig() (*Configuration, error) {
	//TODO implement me
	panic("implement me")
}

func (s contentfulService) SetConfig(config Configuration) error {
	//TODO implement me
	panic("implement me")
}

func NewConfigurationService(repo models.Configuration[Configuration]) Service {
	return &contentfulService{
		repo: repo,
	}
}
