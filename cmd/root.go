package cmd

import (
	"fmt"
	"os"
	"test-cli/internal/config"
	"test-cli/internal/contentful"
	"test-cli/internal/optimizely"
	"test-cli/internal/tui"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "test-cli",
	Short:         "A brief description of your application",
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		styles := tui.AutoStyles()
		errPrefix := styles.Error.Render("Error:")
		_, err := fmt.Fprintf(os.Stderr, "%s %s\n", errPrefix, err.Error())
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(cmd.UsageString())
		return err
	})

	v, err := config.Init()
	if err != nil {
		panic(err)
	}

	cRepo := contentful.NewConfigRepo(v)
	cService := contentful.NewConfigurationService(cRepo)

	opRepo := optimizely.NewConfigurationRepo(v)
	sourceRepo, err := optimizely.NewSourceRepository()
	if err != nil {
		panic(err)
	}
	opService := optimizely.NewOptimizelyService(opRepo, sourceRepo)

	styles := tui.AutoStyles()
	rootCmd.AddCommand(optimizely.NewCommand(opService, styles))
	rootCmd.AddCommand(contentful.NewCommand(cService))
}
