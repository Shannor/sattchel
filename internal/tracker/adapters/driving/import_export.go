package driving

import (
	"fmt"
	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
)

func exportCmd(service *core.Service, writer printer.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "export [filepath]",
		Short: "Export tracker data to a file (defaults to ./tracker.json)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := "tracker.json"
			if len(args) > 0 {
				filePath = args[0]
			}
			err := service.Export(cmd.Context(), filePath)
			if err != nil {
				return err
			}
			writer.Success(fmt.Sprintf("Tracker data successfully exported to %s", filePath))
			return nil
		},
	}
}

func importCmd(service *core.Service, writer printer.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "import [filepath]",
		Short: "Import tracker data from a file (defaults to ./tracker.json)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := "tracker.json"
			if len(args) > 0 {
				filePath = args[0]
			}
			err := service.Import(cmd.Context(), filePath)
			if err != nil {
				return err
			}
			writer.Success(fmt.Sprintf("Tracker data successfully imported from %s", filePath))
			return nil
		},
	}
}
