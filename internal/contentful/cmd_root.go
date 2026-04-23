package contentful

import (
	"github.com/spf13/cobra"
)

func NewCommand(s Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contentful",
		Short:   "Contentful commands",
		Aliases: []string{"ctf"},
		Version: "0.0.1",
	}
	return cmd
}
