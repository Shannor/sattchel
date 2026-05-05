package contentfulcli

import (
	"test-cli/internal/contentful"

	"github.com/spf13/cobra"
)

func NewCommand(s contentful.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contentful",
		Short:   "Contentful commands",
		Aliases: []string{"ctf"},
		Version: "0.0.1",
	}
	return cmd
}
