package optcli

import (
	"context"
	"fmt"
	"test-cli/internal/optimizely"
	"test-cli/internal/tui"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func cmdConfig(service optimizely.Service, styles tui.Styles) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "config",
		Short:        "Manage configs",
		Aliases:      []string{"co"},
		SilenceUsage: true,
	}
	configCmd.AddCommand(set(service))
	configCmd.AddCommand(get(service, styles))
	return configCmd
}

func set(service optimizely.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Long: `Set an allowed configuration value.
	If no key is provided, a list of available keys will be displayed.
   Examples:
     test-cli optimizely config set 
     test-cli optimizely config set apiKey
     test-cli optimizely config set apiKey <value>
     `,
		Args:         cobra.MaximumNArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch len(args) {
			case 0:
				err := noChoiceConfig(cmd.Context(), service)
				if err != nil {
					return fmt.Errorf("failed to set config: %w", err)
				}
			case 1:
			case 2:
			}
			return nil
		},
	}
}

func get(service optimizely.Service, styles tui.Styles) *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Long: `Get an configuration value.
	If no key is provided, all keys will be displayed.
   Examples:
     test-cli optimizely config get 
     test-cli optimizely config get apiKey
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := service.GetConfig(cmd.Context())
			if err != nil {
				return err
			}
			switch len(args) {
			case 0:
				fmt.Println(renderConfig(cfg, styles))
			case 1:
				fmt.Println(renderConfig(cfg, styles))
			default:
				return fmt.Errorf("unsupported amount of commands")
			}
			return nil
		},
	}
}

func noChoiceConfig(ctx context.Context, service optimizely.Service) error {
	choice := ""
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Pick a config to set").
				Options(
					huh.NewOption("API Key", "apiKey"),
					huh.NewOption("Products", "products"),
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
			Placeholder: "Insert Config Value",
		})).Run()
		if err != nil {
			return fmt.Errorf("failed to run program: %w", err)
		}
		if v, ok := value.(tui.InputModel); ok {
			if v.Value() == "" {
				return fmt.Errorf("value cannot be empty")
			}
			c := optimizely.Configuration{APIKey: v.Value()}
			err = service.SetConfig(ctx, c)
			if err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
		}
	case "products":
	}
	return nil
}
