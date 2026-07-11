package driving

import (
	"context"
	"fmt"
	"sattchel/internal/tui"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func cmdConfig(config *Config, styles tui.Styles) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "config",
		Short:        "Manage optimizely configs",
		SilenceUsage: true,
	}
	configCmd.AddCommand(setConfig(config))
	configCmd.AddCommand(getConfig(config, styles))
	return configCmd
}

func setConfig(config *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "set",
		Short: "Set Optimizely configuration values",
		Long: `Set an allowed configuration value.
	Examples:
     satt optimizely config set 
     `,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := noChoiceConfig(cmd.Context(), config)
			if err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
			return nil
		},
	}
}

func getConfig(config *Config, styles tui.Styles) *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get Optimizely configuration values",
		Long: `Get all configuration values.
   Examples:
     satt optimizely config get 
     `,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Get()
			if err != nil {
				return err
			}
			fmt.Println(renderConfig(cfg, styles))
			return nil
		},
	}
}

func noChoiceConfig(ctx context.Context, config *Config) error {
	choice := ""
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Pick a config to set").
				Options(
					huh.NewOption("API Key", "apiKey"),
					huh.NewOption("Cache TTL", "cacheTTLMinutes"),
				).
				Value(&choice),
		).WithShowHelp(true),
	).Run()
	if err != nil {
		return fmt.Errorf("failed to select: %w", err)
	}

	// TODO: Convert to using a full form since it isn't as dynamic
	switch choice {
	case "apiKey":
		value, err := tea.NewProgram(tui.NewTextPrompt(tui.InputConfig{
			Placeholder: "Insert API Key",
		})).Run()
		if err != nil {
			return fmt.Errorf("failed to run program: %w", err)
		}
		if v, ok := value.(tui.InputModel); ok {
			if v.Value() == "" {
				return fmt.Errorf("value cannot be empty")
			}
			_, err = config.Update(func(cfg *Configuration) error {
				cfg.APIKey = v.Value()
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
		}
	case "cacheTTLMinutes":
		value, err := tea.NewProgram(tui.NewTextPrompt(tui.InputConfig{
			Placeholder: "Insert time in minutes",
			Header:      "Provide cache TTL in minutes. ex (10 = 10 minutes)",
		})).Run()
		if err != nil {
			return fmt.Errorf("failed to run program: %w", err)
		}
		if v, ok := value.(tui.InputModel); ok {
			if v.Value() == "" {
				return fmt.Errorf("value cannot be empty")
			}
			_, err = config.Update(func(cfg *Configuration) error {
				v, err := strconv.Atoi(v.Value())
				if err != nil {
					return err
				}
				cfg.CacheTTLMinutes = int64(time.Duration(v) * time.Minute)
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
		}
	}
	return nil
}
