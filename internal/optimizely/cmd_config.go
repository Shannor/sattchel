package optimizely

import (
	"fmt"
	"test-cli/internal/tui"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func cmdConfig(service Service) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:          "config",
		Short:        "Manage configs",
		Aliases:      []string{"co"},
		Version:      "0.0.1",
		SilenceUsage: true,
	}
	configCmd.AddCommand(set(service))
	return configCmd
}

func set(service Service) *cobra.Command {
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
				err := noChoiceConfig(service)
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

func noChoiceConfig(service Service) error {
	choice := ""
	err := huh.NewSelect[string]().
		Title("Pick a config to set").
		Options(
			huh.NewOption("API Key", "apiKey"),
			huh.NewOption("Products", "products"),
			huh.NewOption("Count", "count"),
		).
		Value(&choice).Run()
	if err != nil {
		return fmt.Errorf("failed to select: %w", err)
	}

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
			c, err := service.GetConfig()
			if err != nil {
				return fmt.Errorf("failed to get config: %w", err)

			}
			c.APIKey = v.Value()
			err = service.SetConfig(*c)
			if err != nil {
				return fmt.Errorf("failed to set config: %w", err)
			}
		}
	case "products":
	}
	return nil
}
