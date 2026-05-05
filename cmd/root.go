package cmd

import (
	"fmt"
	"os"
	"test-cli/internal/cli/optcli"
	"test-cli/internal/cli/update"
	"test-cli/internal/config"
	"test-cli/internal/optimizely"
	"test-cli/internal/printer"
	"test-cli/internal/tui"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var updateCh <-chan config.UpdateInformation

var rootCmd = &cobra.Command{
	Use:           "test-cli",
	Short:         "A brief description of your application",
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       config.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		updateCh = config.NewUpdater().CheckForUpdate()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if updateCh == nil {
			return
		}
		if update, ok := <-updateCh; ok {
			writer := printer.NewStyleWriter(tui.AutoStyles())
			if update.NeedToUpdate {
				msg := fmt.Sprintf("A new version is available: %s (current: %s). Run \"test-cli update\" to upgrade.", update.NewVersion, update.CurrentVersion)
				writer.Info(msg)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		styles := tui.AutoStyles()
		writer := printer.NewStyleWriter(styles)
		msg := fmt.Sprintf("%s %s\n", "Error:", err.Error())
		writer.Error(msg)
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

	styles := tui.AutoStyles()
	writer := printer.NewStyleWriter(styles)

	opService := setupOptimizely(v)
	rootCmd.AddCommand(optcli.NewCommand(opService, writer, styles))
	rootCmd.AddCommand(update.NewCommand(writer))
}

func setupOptimizely(v *viper.Viper) optimizely.Service {
	opRepo := optimizely.NewConfigurationRepo(v)
	sourceRepo, err := optimizely.NewSourceRepository()
	if err != nil {
		panic(err)
	}
	cfg, err := opRepo.GetConfig()
	if err != nil {
		panic(err)
	}
	client := optimizely.BaseFlagClient(cfg)
	factory := optimizely.NewFlagsDMFactory(client, cfg.APIKey)
	return optimizely.NewOptimizelyService(opRepo, sourceRepo, factory)
}
