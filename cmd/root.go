package cmd

import (
	"os"
	"test-cli/internal/config"
	"test-cli/internal/contentful"
	"test-cli/internal/optimizely"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	v := viper.New()
	err := config.Init(v)
	if err != nil {
		panic(err)
	}
	cRepo := contentful.NewContentfulRepository(v)
	opRepo := optimizely.NewConfigRepo(v)

	cService := contentful.NewConfigurationService(cRepo)
	opService := optimizely.NewOptimizelyService(opRepo)

	rootCmd.AddCommand(optimizely.NewCommand(opService))
	rootCmd.AddCommand(contentful.NewCommand(cService))
}
