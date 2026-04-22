/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package configs

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

// Setup reads the config file and initializes the Config instance
func Setup() error {
	// Add search paths to find the file
	viper.AddConfigPath("$HOME/.config/test-cli")
	viper.SetConfigName(FileName)
	viper.SetConfigType(FileType)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			return initConfig()
		}
		// Config file was found but another error was produced
		fmt.Println("Config file found but another error was produced")
		return err
	}
	return nil
}

// Get returns the Config representation of the config file
// Technically, more data can be in the file, but only the Config Struct is used
func Get() (Config, error) {
	c := Config{}
	err := viper.Unmarshal(&c)
	if err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return c, nil
}

func initConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	fullPath := fmt.Sprintf("%s/%s/%s.%s", home, SubFolder, FileName, FileType)

	err = os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create config folder: %w", err)
	}

	if file, err := os.Create(fullPath); err != nil { // perm 0666
		return fmt.Errorf("failed to create config file: %w", err)
	} else {
		defer file.Close()
	}

	err = viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}
