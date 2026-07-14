package loader

import (
	"os"
	"time"

	"charm.land/huh/v2/spinner"
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
		return spinner.New().
			Title(title).
			Action(func() {
				<-done
			}).Run()
	}
}
