package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"sattchel/internal/printer"
	"testing"

	"github.com/spf13/viper"
)

func TestThemeCmd(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")

	if err := os.WriteFile(configPath, []byte("theme: default\n"), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	originalV := v
	defer func() { v = originalV }()

	v = viper.New()
	v.SetConfigFile(configPath)
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	buf := new(bytes.Buffer)
	writer := printer.NewStyleWriterWithWriters(buf, buf)
	cmd := newThemeCmd(writer)

	if cmd.Use != "theme" {
		t.Errorf("expected use 'theme', got %q", cmd.Use)
	}

	// Verify command definition
	if len(cmd.Commands()) != 0 {
		t.Errorf("expected 0 subcommands, got %d", len(cmd.Commands()))
	}
}
