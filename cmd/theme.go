package cmd

import (
	"fmt"
	"sattchel/internal/printer"
	"sattchel/internal/tui"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func newThemeCmd(writer printer.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "theme",
		Short: "Select and set the active CLI theme",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var selectedTheme string

			err := tui.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select CLI Theme").
						Options(
							huh.NewOption("Default", "default"),
							huh.NewOption("Gruvbox", "gruvbox"),
						).
						Value(&selectedTheme).
						Height(4),
				),
			).Run()
			if err != nil {
				return err
			}

			if selectedTheme == "" {
				return nil
			}

			// Save to viper config
			v.Set("theme", selectedTheme)
			if err := v.WriteConfig(); err != nil {
				return fmt.Errorf("failed to save theme config: %w", err)
			}

			tui.SetTheme(selectedTheme)
			writer.Success(fmt.Sprintf("Theme successfully set to %s", selectedTheme))
			return nil
		},
	}
}
