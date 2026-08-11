package driving

import (
	"context"
	"fmt"
	"sattchel/internal/printer"
	"sattchel/internal/tui"
	"strconv"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func configCmd(config *Config, writer printer.Writer) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "config",
		Short:        "Manage optimizely configs",
		SilenceUsage: true,
	}
	configCmd.AddCommand(setConfig(config, writer))
	configCmd.AddCommand(getConfig(config))
	return configCmd
}

func setConfig(config *Config, writer printer.Writer) *cobra.Command {
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
			err := noChoiceConfig(cmd.Context(), config, writer)
			if err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
			return nil
		},
	}
}

func getConfig(config *Config) *cobra.Command {
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
			styles := tui.AutoStyles()
			fmt.Println(renderConfig(cfg, styles))
			return nil
		},
	}
}

func noChoiceConfig(ctx context.Context, config *Config, writer printer.Writer) error {
	choice := ""
	err := tui.NewForm(
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

	var apiKey string
	var cacheTTL string

	switch choice {
	case "apiKey":
		err := tui.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("API Key").
					Placeholder("Insert API Key").
					Value(&apiKey).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("API Key cannot be empty")
						}
						return nil
					}),
			),
		).Run()
		if err != nil {
			return err
		}

		_, err = config.Update(func(cfg *Configuration) error {
			cfg.APIKey = apiKey
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to set config: %w", err)
		}
		writer.Success("API Key updated successfully")

	case "cacheTTLMinutes":
		err := tui.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Provide cache TTL in minutes (e.g. 10)").
					Placeholder("Insert time in minutes").
					Value(&cacheTTL).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("cache TTL cannot be empty")
						}
						val, err := strconv.Atoi(s)
						if err != nil || val <= 0 {
							return fmt.Errorf("must be a positive integer")
						}
						return nil
					}),
			),
		).Run()
		if err != nil {
			return err
		}

		val, _ := strconv.Atoi(cacheTTL)
		_, err = config.Update(func(cfg *Configuration) error {
			cfg.CacheTTLMinutes = int64(time.Duration(val) * time.Minute)
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to set config: %w", err)
		}
		writer.Success(fmt.Sprintf("Cache TTL updated successfully to %d minutes", val))
	}
	return nil
}
