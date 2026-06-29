package driving

import (
	"context"
	"fmt"
	"sattchel/internal/optimizely/core"
	"sattchel/internal/tui"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func cmdConfig(service *core.Service, styles tui.Styles) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "config",
		Short:        "Manage configs",
		Aliases:      []string{"co"},
		SilenceUsage: true,
	}
	configCmd.AddCommand(setConfig(service))
	configCmd.AddCommand(getConfig(service, styles))
	return configCmd
}

func setConfig(service *core.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Long: `Set an allowed configuration value.
	If no key is provided, a list of available keys will be displayed.
   Examples:
     sattchel optimizely config set 
     sattchel optimizely config set apiKey
     sattchel optimizely config set apiKey <value>
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

func getConfig(service *core.Service, styles tui.Styles) *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Long: `Get an configuration value.
	If no key is provided, all keys will be displayed.
   Examples:
     sattchel optimizely config get 
     sattchel optimizely config get apiKey
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

func noChoiceConfig(ctx context.Context, service *core.Service) error {
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
			c := core.Configuration{APIKey: v.Value()}
			err = service.SetConfig(ctx, c)
			if err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
		}
	case "products":
	}
	return nil
}
