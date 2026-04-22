package configs

type ConfigService interface {
	GetConfig() (*Config, error)
	SetConfig(config Config) error
	Init() error
}

type configService struct {
	ConfigRepo ConfigRepo
}

func NewConfigService(repo ConfigRepo) ConfigService {
	return &configService{
		ConfigRepo: repo,
	}
}
