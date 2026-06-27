package cmd

import (
	"context"
	"fmt"
	"os"
	"sattchel/internal/cli/optcli"
	"sattchel/internal/cli/update"
	"sattchel/internal/config"
	"sattchel/internal/optimizely"
	"sattchel/internal/printer"
	"sattchel/internal/tui"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var updateCh <-chan config.UpdateInformation

var rootCmd = &cobra.Command{
	Use:           "sattchel",
	Short:         "A collection of tools for optimizing my workflows or fun",
	SilenceErrors: true,
	SilenceUsage:  true,
	Aliases:       []string{"sat", "satt"},
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
				msg := fmt.Sprintf("A new version is available: %s (current: %s). Run \"sattchel update\" to upgrade.", update.NewVersion, update.CurrentVersion)
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
	opRepo := optimizely.NewConfigDM(v)
	cfg, err := opRepo.Get(context.Background(), "")
	if err != nil {
		panic(err)
	}
	client := optimizely.BaseFlagClient(cfg)
	v2Client := optimizely.BaseV2Client(cfg)
	factory := optimizely.NewFlagsDMFactory(client, cfg.APIKey)
	envFactory := optimizely.NewEnvironmentsDMFactory(v2Client, cfg.APIKey)
	projectDM := optimizely.NewProjectsDM(v2Client)
	return optimizely.NewOptimizelyService(opRepo, projectDM, factory, envFactory)
}
