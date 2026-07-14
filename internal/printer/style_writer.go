package printer

import (
	"fmt"
	"os"
	"sattchel/internal/tui"
	"sync"
)

type Writer interface {
	Info(msg string)
	Success(msg string)
	Error(msg string)
}

type StyleWriter struct {
	mu        sync.Mutex
	styles    tui.Styles
	hasStyles bool
	lazy      bool
}

// NewStyleWriter returns a new StyleWriter. If no styles are provided,
// it will load the terminal styles lazily on the first print action.
func NewStyleWriter(styles ...tui.Styles) Writer {
	if len(styles) > 0 {
		return &StyleWriter{
			styles:    styles[0],
			hasStyles: true,
		}
	}
	return &StyleWriter{
		lazy: true,
	}
}

func (w *StyleWriter) getStyles() tui.Styles {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lazy && !w.hasStyles {
		w.styles = tui.AutoStyles()
		w.hasStyles = true
	}
	return w.styles
}

func (w *StyleWriter) Info(msg string) {
	styles := w.getStyles()
	fmt.Fprintln(os.Stdout, styles.Info.Render(msg))
}

func (w *StyleWriter) Success(msg string) {
	styles := w.getStyles()
	fmt.Fprintln(os.Stdout, styles.Success.Render(msg))
}

func (w *StyleWriter) Error(msg string) {
	styles := w.getStyles()
	// TODO: Add a check for if the message starts with Error/error and add it if not
	fmt.Fprintln(os.Stderr, styles.Error.Render(msg))
}
