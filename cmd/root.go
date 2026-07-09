package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sattchel/internal/cli/update"
	"sattchel/internal/config"
	optimizelyDriven "sattchel/internal/optimizely/adapters/driven"
	optimizelyDriving "sattchel/internal/optimizely/adapters/driving"
	optimizelyCore "sattchel/internal/optimizely/core"
	"sattchel/internal/printer"
	trackerDriven "sattchel/internal/tracker/adapters/driven"
	trackerDriving "sattchel/internal/tracker/adapters/driving"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var updateCh <-chan config.UpdateInformation

const defaultBinaryName = "sattchel"

var rootCmd = &cobra.Command{
	Use:           defaultBinaryName,
	Short:         "A collection of tools for optimizing my workflows or fun",
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       config.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if isCompletionCommand(cmd) {
			updateCh = nil
			return
		}
		updateCh = config.NewUpdater().CheckForUpdate()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if isCompletionCommand(cmd) {
			return
		}
		if updateCh == nil {
			return
		}
		if update, ok := <-updateCh; ok {
			writer := printer.NewStyleWriter(tui.AutoStyles())
			if update.NeedToUpdate {
				msg := fmt.Sprintf(
					"A new version is available: %s (current: %s). Run \"%s update\" to upgrade.",
					update.NewVersion,
					update.CurrentVersion,
					cmd.Root().Name(),
				)
				writer.Info(msg)
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.Use = executableName()
	if err := config.EnsureExecutableAliases(); err != nil {
		slog.Warn("failed to sync cli aliases", slog.String("error", err.Error()))
	}
	err := rootCmd.Execute()
	if err != nil {
		styles := tui.AutoStyles()
		writer := printer.NewStyleWriter(styles)
		msg := fmt.Sprintf("%s %s\n", "Error:", err.Error())
		writer.Error(msg)
		os.Exit(1)
	}
}

func executableName() string {
	name := filepath.Base(os.Args[0])
	if name == "." || name == string(filepath.Separator) || name == "" {
		return defaultBinaryName
	}
	return name
}

func isCompletionCommand(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "completion", "__complete", "__completeNoDesc":
			return true
		}
	}
	return false
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

	rootCmd.AddCommand(setupTracker(v))
	rootCmd.AddCommand(optimizelyDriving.NewCommand(opService, v, writer, styles))
	rootCmd.AddCommand(update.NewCommand(writer))
}

func setupTracker(v *viper.Viper) *cobra.Command {
	path := filepath.Join(config.ResolvedConfigDir, "tracker.json")
	fileStorage := trackerDriven.NewFileStorage(path, nil)
	trackerService := core.NewService(fileStorage)
	return trackerDriving.NewCommand(trackerService, v)
}

func setupOptimizely(v *viper.Viper) *optimizelyCore.Service {
	var cfg optimizelyDriving.Configuration
	_ = v.UnmarshalKey("optimizely", &cfg)

	client := optimizelyDriven.BaseFlagClient(cfg.APIKey)
	v2Client := optimizelyDriven.BaseV2Client(cfg.APIKey)
	factory := optimizelyDriven.NewFlagsDMFactory(client, cfg.APIKey)
	envFactory := optimizelyDriven.NewEnvironmentsDMFactory(v2Client, cfg.APIKey)
	projectDM := optimizelyDriven.NewProjectsDM(v2Client)
	return optimizelyCore.NewService(projectDM, factory, envFactory)
}
