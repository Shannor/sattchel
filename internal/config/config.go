package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const SubFolder = ".config/satt"
const FileName = "config"
const FileType = "yml"

var ResolvedConfigDir string

func Init() (*viper.Viper, error) {
	// Read local .env file using a temporary Viper instance
	envViper := viper.New()
	envViper.SetConfigFile(".env")
	envViper.SetConfigType("env")
	_ = envViper.ReadInConfig() // Ignore error if .env doesn't exist

	v := viper.New()

	configDir := envViper.GetString("SATT_CONFIG_DIR")
	if configDir == "" {
		configDir = os.Getenv("SATT_CONFIG_DIR")
	}

	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(home, SubFolder)
	}

	ResolvedConfigDir = configDir

	v.AddConfigPath(configDir)
	v.SetConfigName(FileName)
	v.SetConfigType(FileType)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			fullPath := filepath.Join(configDir, fmt.Sprintf("%s.%s", FileName, FileType))

			err = os.MkdirAll(filepath.Dir(fullPath), 0755)
			if err != nil {
				return nil, fmt.Errorf("failed to create config folder: %w", err)
			}

			if file, err := os.Create(fullPath); err != nil { // perm 0666
				return nil, fmt.Errorf("failed to create config file: %w", err)
			} else {
				defer file.Close()
			}

			err = v.WriteConfig()
			if err != nil {
				return nil, fmt.Errorf("failed to write config: %w", err)
			}
			return v, nil
		}
		return nil, err
	}
	return v, nil
}
