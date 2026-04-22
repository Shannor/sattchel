package config

import (
	"fmt"
	"strings"
	"test-cli/internal/tui"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func set() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Long: `Set a configuration value using dot notation.
   Examples:
     mycli config set optimizely.apiKey 
     mycli config set contentful.spaceId`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateKey(args[0]); err != nil {
				return err
			}
			value, err := tea.NewProgram(tui.NewTextPrompt(tui.InputConfig{
				Placeholder: "Insert Config Value",
			})).Run()
			if err != nil {
				return fmt.Errorf("failed to run program: %w", err)
			}
			if v, ok := value.(tui.InputModel); ok {
				viper.Set(args[0], v.Value())
			}

			if err := viper.WriteConfig(); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
			return nil
		},
	}
}
func validateKey(key string) error {
	allowedPrefixes := map[string]bool{
		"optimizely": true,
		"contentful": true,
	}

	prefix := strings.Split(key, ".")[0]

	if !allowedPrefixes[prefix] {
		return fmt.Errorf("invalid config section '%s'. Allowed sections: optimizely, contentful", prefix)
	}
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	return nil
}
