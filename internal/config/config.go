package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const SubFolder = ".config/test-cli"
const FileName = "config"
const FileType = "yml"

func Init() (*viper.Viper, error) {
	v := viper.New()
	v.AddConfigPath("$HOME/.config/test-cli")
	v.SetConfigName(FileName)
	v.SetConfigType(FileType)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			fullPath := filepath.Join(home, SubFolder, fmt.Sprintf("%s.%s", FileName, FileType))

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
