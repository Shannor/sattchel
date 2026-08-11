package driving

import (
	"fmt"
	"os"
	"path/filepath"
	"sattchel/internal/config"
	"sattchel/internal/printer"

	"github.com/spf13/cobra"
)

func cache(cfgStore *Config, writer printer.Writer) *cobra.Command {
	var cacheCmd = &cobra.Command{
		Use:          "cache",
		Short:        "Manage Optimizely cache",
		SilenceUsage: true,
	}

	cacheCmd.AddCommand(clearCache(cfgStore, writer))
	return cacheCmd
}

func clearCache(cfgStore *Config, writer printer.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Wipe all local Optimizely cache files from disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern1 := filepath.Join(config.ResolvedConfigDir, "optimizely_cache_*.json")
			pattern2 := filepath.Join(config.ResolvedConfigDir, "optimizely_cache.json")

			files1, err := filepath.Glob(pattern1)
			if err != nil {
				return fmt.Errorf("failed to search cache files: %w", err)
			}
			files2, err := filepath.Glob(pattern2)
			if err != nil {
				return fmt.Errorf("failed to search cache files: %w", err)
			}

			allFiles := append(files1, files2...)
			if len(allFiles) == 0 {
				writer.Info("No Optimizely cache files found to clear.")
				return nil
			}

			deletedCount := 0
			for _, file := range allFiles {
				if err := os.Remove(file); err == nil {
					deletedCount++
					writer.Success(fmt.Sprintf("Cleared cache file: %s", filepath.Base(file)))
				} else if !os.IsNotExist(err) {
					writer.Warn(fmt.Sprintf("Warning: failed to delete cache file %s: %v", file, err))
				}
			}

			writer.Success(fmt.Sprintf("Successfully cleared %d local cache file(s).", deletedCount))
			return nil
		},
	}
}
