package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureExecutableAliasesCreatesAliases(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	executablePath := filepath.Join(tempDir, binaryFileName(repoName, runtime.GOOS))
	if err := os.WriteFile(executablePath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	if err := ensureExecutableAliases(executablePath, runtime.GOOS); err != nil {
		t.Fatalf("ensure aliases: %v", err)
	}

	for _, name := range []string{"sat", "satt"} {
		aliasPath := filepath.Join(tempDir, binaryFileName(name, runtime.GOOS))
		info, err := os.Lstat(aliasPath)
		if err != nil {
			t.Fatalf("stat alias %s: %v", name, err)
		}

		if runtime.GOOS == "windows" {
			if info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("expected %s to be a copied executable on windows", name)
			}
			continue
		}

		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("expected %s to be a symlink", name)
		}

		target, err := os.Readlink(aliasPath)
		if err != nil {
			t.Fatalf("readlink %s: %v", name, err)
		}
		if target != repoName {
			t.Fatalf("expected %s to point to %s, got %s", name, repoName, target)
		}
	}
}

func TestEnsureExecutableAliasesRepairsStaleSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior is unix-specific")
	}
	t.Parallel()

	tempDir := t.TempDir()
	executablePath := filepath.Join(tempDir, repoName)
	if err := os.WriteFile(executablePath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	staleTarget := filepath.Join(tempDir, "old-binary")
	if err := os.WriteFile(staleTarget, []byte("old"), 0o755); err != nil {
		t.Fatalf("write stale target: %v", err)
	}

	staleAlias := filepath.Join(tempDir, "sat")
	if err := os.Symlink(filepath.Base(staleTarget), staleAlias); err != nil {
		t.Fatalf("create stale symlink: %v", err)
	}

	if err := ensureExecutableAliases(executablePath, runtime.GOOS); err != nil {
		t.Fatalf("ensure aliases: %v", err)
	}

	target, err := os.Readlink(staleAlias)
	if err != nil {
		t.Fatalf("read repaired symlink: %v", err)
	}
	if target != repoName {
		t.Fatalf("expected repaired symlink to point to %s, got %s", repoName, target)
	}
}

func TestEnsureExecutableAliasesSkipsUnknownBinaryOutsideInstallDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	executablePath := filepath.Join(tempDir, binaryFileName("custom-name", runtime.GOOS))
	if err := os.WriteFile(executablePath, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	if err := ensureExecutableAliases(executablePath, runtime.GOOS); err != nil {
		t.Fatalf("ensure aliases: %v", err)
	}

	for _, name := range supportedBinaryNames {
		if name == repoName {
			continue
		}
		aliasPath := filepath.Join(tempDir, binaryFileName(name, runtime.GOOS))
		if _, err := os.Lstat(aliasPath); !os.IsNotExist(err) {
			t.Fatalf("expected no alias at %s, got err=%v", aliasPath, err)
		}
	}
}
