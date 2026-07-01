package driving

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	stateFileName = "tracker_state"
	stateFileType = "yml"
	subFolder     = ".config/sattchel"
)

type Config struct {
	v *viper.Viper
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	configDir := filepath.Join(home, subFolder)
	v.AddConfigPath(configDir)
	v.SetConfigName(stateFileName)
	v.SetConfigType(stateFileType)

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return nil, fmt.Errorf("create config dir: %w", err)
			}
			filePath := filepath.Join(configDir, fmt.Sprintf("%s.%s", stateFileName, stateFileType))
			file, err := os.Create(filePath)
			if err != nil {
				return nil, fmt.Errorf("create config file: %w", err)
			}
			file.Close()

			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("read created config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	return &Config{v: v}, nil
}

func (c *Config) CurrentProjectID() string {
	return c.v.GetString("current_project_id")
}

func (c *Config) SetCurrentProjectID(id string) error {
	c.v.Set("current_project_id", id)
	return c.v.WriteConfig()
}

func (c *Config) CurrentGoalID() string {
	return c.v.GetString("current_goal_id")
}

func (c *Config) SetCurrentGoalID(id string) error {
	c.v.Set("current_goal_id", id)
	return c.v.WriteConfig()
}

func (c *Config) ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, subFolder), nil
}
