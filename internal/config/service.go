package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const SubFolder = ".config/test-cli"
const FileName = "config"
const FileType = "yml"

type configServiceKey struct{}

type Service interface {
	Init() error
	GetOptimizelyRepo() Repository[OptimizelyConfig]
	GetContentfulRepo() Repository[ContentfulConfig]
	WithContext(ctx context.Context) context.Context
	FromContext(ctx context.Context) Service
}

type service struct {
	v *viper.Viper
}

var _ Service = (*service)(nil)

func NewConfigurationService() Service {
	// TODO: Revisit injecting vs DI
	return &service{
		v: viper.New(),
	}
}

func (s *service) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, configServiceKey{}, s)
}

func (s *service) FromContext(ctx context.Context) Service {
	return ctx.Value(configServiceKey{}).(Service)
}

func (s *service) Init() error {
	s.v.AddConfigPath("$HOME/.config/test-cli")
	s.v.SetConfigName(FileName)
	s.v.SetConfigType(FileType)

	fmt.Println("Looking for config file...")
	if err := s.v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			fullPath := filepath.Join(home, SubFolder, fmt.Sprintf("%s.%s", FileName, FileType))

			err = os.MkdirAll(filepath.Dir(fullPath), 0755)
			if err != nil {
				return fmt.Errorf("failed to create config folder: %w", err)
			}

			if file, err := os.Create(fullPath); err != nil { // perm 0666
				return fmt.Errorf("failed to create config file: %w", err)
			} else {
				defer file.Close()
			}

			err = s.v.WriteConfig()
			if err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
			return nil
		}
		// Config file was found but another error was produced
		fmt.Println("Config file found but another error was produced")
		return err
	}
	return nil
}

func (s *service) GetOptimizelyRepo() Repository[OptimizelyConfig] {
	return &OptimizelyRepository{v: s.v}
}

func (s *service) GetContentfulRepo() Repository[ContentfulConfig] {
	return &ContentfulRepository{v: s.v}
}
