package loader

import (
	"os"
	"time"

	"sattchel/internal/tui"

	"charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-isatty"
)

const DefaultThreshold = 100 * time.Millisecond

// IsTerminal returns true if standard output is a TTY.
func IsTerminal() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

func Run(title string, action func()) error {
	return RunWithThreshold(title, DefaultThreshold, action)
}

func RunWithThreshold(title string, threshold time.Duration, action func()) error {
	if !IsTerminal() {
		action()
		return nil
	}

	done := make(chan struct{})
	go func() {
		action()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(threshold):
		s := tui.AutoStyles()
		spinnerTheme := spinner.ThemeFunc(func(isDark bool) *spinner.Styles {
			return &spinner.Styles{
				Spinner: lipgloss.NewStyle().Foreground(s.Success.GetForeground()),
				Title:   lipgloss.NewStyle().Foreground(s.Text.GetForeground()),
			}
		})
		return spinner.New().
			Title(title).
			WithTheme(spinnerTheme).
			Action(func() {
				<-done
			}).Run()
	}
}
