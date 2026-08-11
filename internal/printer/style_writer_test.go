package printer

import (
	"bytes"
	"sattchel/internal/tui"
	"strings"
	"testing"
)

func TestStyleWriter_LazyInitialization(t *testing.T) {
	// Clean up theme selection at end of test
	defer tui.SetTheme("default")

	var outBuf, errBuf bytes.Buffer
	writer := NewStyleWriterWithWriters(&outBuf, &errBuf)

	// Since we did not pass explicit styles, it should lazy-load styles on the first print action.
	writer.Info("hello info")

	output := outBuf.String()
	if !strings.Contains(output, "hello info") {
		t.Errorf("expected output to contain %q, got %q", "hello info", output)
	}
}

func TestStyleWriter_Methods(t *testing.T) {
	defer tui.SetTheme("default")

	for _, theme := range []string{"default", "gruvbox"} {
		t.Run("theme_"+theme, func(t *testing.T) {
			tui.SetTheme(theme)

			var outBuf, errBuf bytes.Buffer
			// Lazy load the styles based on the active theme
			writer := NewStyleWriterWithWriters(&outBuf, &errBuf)

			// Info
			outBuf.Reset()
			writer.Info("info-msg")
			infoOut := outBuf.String()
			if !strings.Contains(infoOut, "info-msg") {
				t.Errorf("Info() did not print message, got %q", infoOut)
			}

			// Success
			outBuf.Reset()
			writer.Success("success-msg")
			successOut := outBuf.String()
			if !strings.Contains(successOut, "success-msg") {
				t.Errorf("Success() did not print message, got %q", successOut)
			}

			// Warn
			outBuf.Reset()
			writer.Warn("warn-msg")
			warnOut := outBuf.String()
			if !strings.Contains(warnOut, "warn-msg") {
				t.Errorf("Warn() did not print message, got %q", warnOut)
			}

			// Error
			errBuf.Reset()
			writer.Error("error-msg")
			errOut := errBuf.String()
			if !strings.Contains(errOut, "error-msg") {
				t.Errorf("Error() did not print message, got %q", errOut)
			}
		})
	}
}

func TestStyleWriter_ExplicitStyles(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	styles := tui.DefaultStyles(true)
	writer := NewStyleWriterWithWriters(&outBuf, &errBuf, styles)

	writer.Info("hello explicit")

	output := outBuf.String()
	if !strings.Contains(output, "hello explicit") {
		t.Errorf("expected output to contain %q, got %q", "hello explicit", output)
	}
}
