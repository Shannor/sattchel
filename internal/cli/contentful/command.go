package contentful

import (
	"test-cli/internal/config"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contentful",
		Short:   "Contentful commands",
		Aliases: []string{"ctf"},
		Version: "0.0.1",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			svc := config.NewConfigurationService()
			err := svc.Init()
			if err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
