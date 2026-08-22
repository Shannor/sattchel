package cmd

import (
	"fmt"
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

	"time"

	"charm.land/log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var updateCh <-chan config.UpdateInformation
var verbose bool
var v *viper.Viper

// defaultTTL the amount of time we'll hold the cache locally
const defaultTTL = 1 * time.Hour

var rootCmd = &cobra.Command{
	Use:           "satt",
	Short:         "A collection of tools for optimizing my workflows or fun",
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       config.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logLevel := log.WarnLevel
		if verbose {
			logLevel = log.DebugLevel
		}

		log.SetOutput(os.Stderr)
		log.SetLevel(logLevel)
		log.SetReportTimestamp(false)
		log.SetReportCaller(verbose)

		setupTheme(v)
		if isCompletionCommand(cmd) {
			updateCh = nil
			return
		}
		if !isUpdateCommand(cmd) {
			updateCh = config.NewUpdater().CheckForUpdate()
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if isCompletionCommand(cmd) {
			return
		}
		if updateCh == nil {
			return
		}
		if update, ok := <-updateCh; ok {
			if update.NeedToUpdate {
				writer := printer.NewStyleWriter(tui.AutoStyles())
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
	err := rootCmd.Execute()
	if err != nil {
		styles := tui.AutoStyles()
		writer := printer.NewStyleWriter(styles)
		msg := fmt.Sprintf("%s %s\n", "Error:", err.Error())
		writer.Error(msg)
		os.Exit(1)
	}
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

func isUpdateCommand(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "update":
			return true
		}
	}
	return false
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.Println(cmd.UsageString())
		return err
	})

	var err error
	v, err = config.Init()
	if err != nil {
		panic(err)
	}

	setupTheme(v)
	writer := printer.NewStyleWriter()
	opService := setupOptimizely(v)

	// Register primary commands
	rootCmd.AddCommand(setupTracker(v, writer))
	rootCmd.AddCommand(optimizelyDriving.NewCommand(opService, v, writer))
	rootCmd.AddCommand(update.NewCommand(writer))
	rootCmd.AddCommand(newThemeCmd(writer))
}

func setupTheme(v *viper.Viper) {
	theme, mode := getModes(v)
	tui.SetTheme(theme)
	tui.SetColorMode(mode)
}

func setupTracker(v *viper.Viper, writer printer.Writer) *cobra.Command {
	path := filepath.Join(config.ResolvedConfigDir, "tracker.json")
	fileStorage := trackerDriven.NewFileStorage(path, nil)
	trackerService := core.NewService(fileStorage)
	return trackerDriving.NewCommand(trackerService, v, writer)
}

func getModes(v *viper.Viper) (string, string) {
	theme := ""
	mode := ""
	if v != nil {
		theme = v.GetString("theme")
		mode = v.GetString("color_mode")
		if mode == "" {
			mode = v.GetString("mode")
		}
	}
	if theme == "" {
		theme = os.Getenv("SATT_THEME")
	}
	if mode == "" {
		mode = os.Getenv("SATT_COLOR_MODE")
		if mode == "" {
			mode = os.Getenv("SATT_MODE")
		}
	}
	return theme, mode
}

func setupOptimizely(v *viper.Viper) *optimizelyCore.Service {
	var cfg optimizelyDriving.Configuration
	_ = v.UnmarshalKey("optimizely", &cfg)

	client := optimizelyDriven.BaseFlagClient(cfg.APIKey)
	v2Client := optimizelyDriven.BaseV2Client(cfg.APIKey)
	factory := optimizelyDriven.NewFlagsDMFactory(client, cfg.APIKey)

	cachePath := filepath.Join(config.ResolvedConfigDir, "optimizely_cache.json")
	ttl := defaultTTL
	if cfg.CacheTTLMinutes > 0 {
		ttl = time.Duration(cfg.CacheTTLMinutes) * time.Minute
	}
	cachedFactory := optimizelyDriven.NewCachedFlagsFactory(factory, cachePath, ttl)

	envFactory := optimizelyDriven.NewEnvironmentsDMFactory(v2Client, cfg.APIKey)
	projectDM := optimizelyDriven.NewProjectsDM(v2Client)
	return optimizelyCore.NewService(projectDM, cachedFactory, envFactory)
}
