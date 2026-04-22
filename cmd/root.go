package cmd

import (
	"os"
	"test-cli/internal/cli/flags"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "test-cli",
	Short: "A brief description of your application",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//Run: func(cmd *cobra.Command, args []string) {},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	//err := configs.Setup()
	//if err != nil {
	//	panic(err)
	//}

	rootCmd.AddCommand(flags.NewCommand())
}
