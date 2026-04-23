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

func Init(v *viper.Viper) error {
	if v == nil {
		panic("viper instance is nil")
	}
	v.AddConfigPath("$HOME/.config/test-cli")
	v.SetConfigName(FileName)
	v.SetConfigType(FileType)

	if err := v.ReadInConfig(); err != nil {
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

			err = v.WriteConfig()
			if err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
			return nil
		}
		return err
	}
	return nil
}
