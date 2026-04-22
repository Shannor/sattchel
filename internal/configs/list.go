package configs

import (
	"fmt"
	"test-cli/internal/printer"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func all() *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "See all configs",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Config{}
			err := viper.Unmarshal(&c)
			if err != nil {
				return fmt.Errorf("failed to unmarshal config: %w", err)
			}
			printer.PrettyPrintColor(c)
			return nil
		},
	}
}
