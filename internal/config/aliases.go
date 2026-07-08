package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var supportedBinaryNames = []string{repoName, "sat", "satt"}

func EnsureExecutableAliases() error {
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate executable: %w", err)
	}
	return ensureExecutableAliases(executablePath, runtime.GOOS)
}

func ensureExecutableAliases(executablePath, goos string) error {
	executablePath, err := filepath.Abs(executablePath)
	if err != nil {
		return fmt.Errorf("failed to normalize executable path: %w", err)
	}

	installDir := filepath.Dir(executablePath)
	executableName := stripExecutableExt(filepath.Base(executablePath), goos)
	primaryPath := filepath.Join(installDir, binaryFileName(repoName, goos))

	if !isSupportedBinaryName(executableName) {
		if _, err := os.Stat(primaryPath); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to inspect primary binary: %w", err)
		}
	}

	targetPath := executablePath
	if _, err := os.Stat(primaryPath); err == nil {
		targetPath = primaryPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect primary binary: %w", err)
	}

	targetName := filepath.Base(targetPath)
	for _, name := range supportedBinaryNames {
		aliasPath := filepath.Join(installDir, binaryFileName(name, goos))
		if aliasPath == targetPath {
			continue
		}
		if err := ensureAlias(targetPath, targetName, aliasPath, goos); err != nil {
			return fmt.Errorf("failed to sync %s alias: %w", name, err)
		}
	}

	return nil
}

func ensureAlias(targetPath, targetName, aliasPath, goos string) error {
	info, err := os.Lstat(aliasPath)
	if err == nil {
		if goos != "windows" && info.Mode()&os.ModeSymlink != 0 {
			matches, err := symlinkMatchesTarget(aliasPath, targetPath)
			if err != nil {
				return err
			}
			if matches {
				return nil
			}
			if err := os.Remove(aliasPath); err != nil {
				return fmt.Errorf("failed to replace stale alias: %w", err)
			}
		} else {
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect alias: %w", err)
	}

	if goos == "windows" {
		return copyFile(targetPath, aliasPath)
	}

	return os.Symlink(targetName, aliasPath)
}

func symlinkMatchesTarget(aliasPath, targetPath string) (bool, error) {
	linkTarget, err := os.Readlink(aliasPath)
	if err != nil {
		return false, fmt.Errorf("failed to read alias target: %w", err)
	}

	if filepath.Clean(linkTarget) == filepath.Base(targetPath) || filepath.Clean(linkTarget) == filepath.Clean(targetPath) {
		return true, nil
	}

	if !filepath.IsAbs(linkTarget) {
		linkTarget = filepath.Join(filepath.Dir(aliasPath), linkTarget)
	}

	return filepath.Clean(linkTarget) == filepath.Clean(targetPath), nil
}

func copyFile(src, dst string) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source binary: %w", err)
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source binary: %w", err)
	}
	defer source.Close()

	destination, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, sourceInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("failed to create alias binary: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("failed to copy alias binary: %w", err)
	}

	return nil
}

func binaryFileName(name, goos string) string {
	if goos == "windows" {
		return name + ".exe"
	}
	return name
}

func stripExecutableExt(name, goos string) string {
	if goos == "windows" {
		return strings.TrimSuffix(name, ".exe")
	}
	return name
}

func isSupportedBinaryName(name string) bool {
	for _, supportedName := range supportedBinaryNames {
		if name == supportedName {
			return true
		}
	}
	return false
}
