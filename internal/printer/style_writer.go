package printer

import (
	"fmt"
	"io"
	"os"
	"sattchel/internal/tui"
	"sync"
)

type Writer interface {
	Info(msg string)
	Success(msg string)
	Error(msg string)
	Warn(msg string)
}

type StyleWriter struct {
	mu        sync.Mutex
	styles    tui.Styles
	hasStyles bool
	lazy      bool
	out       io.Writer
	err       io.Writer
}

// NewStyleWriter returns a new StyleWriter. If no styles are provided,
// it will load the terminal styles lazily on the first print action.
func NewStyleWriter(styles ...tui.Styles) Writer {
	return NewStyleWriterWithWriters(os.Stdout, os.Stderr, styles...)
}

// NewStyleWriterWithWriters returns a new StyleWriter targeting the specified io.Writers.
func NewStyleWriterWithWriters(out, err io.Writer, styles ...tui.Styles) *StyleWriter {
	if len(styles) > 0 {
		return &StyleWriter{
			styles:    styles[0],
			hasStyles: true,
			out:       out,
			err:       err,
		}
	}
	return &StyleWriter{
		lazy: true,
		out:  out,
		err:  err,
	}
}

func (w *StyleWriter) getOut() io.Writer {
	if w.out != nil {
		return w.out
	}
	return os.Stdout
}

func (w *StyleWriter) getErr() io.Writer {
	if w.err != nil {
		return w.err
	}
	return os.Stderr
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
	_, _ = fmt.Fprintln(w.getOut(), styles.Info.Render(msg))
}

func (w *StyleWriter) Warn(msg string) {
	styles := w.getStyles()
	_, _ = fmt.Fprintln(w.getOut(), styles.Warning.Render(msg))
}

func (w *StyleWriter) Success(msg string) {
	styles := w.getStyles()
	_, _ = fmt.Fprintln(w.getOut(), styles.Success.Render(msg))
}

func (w *StyleWriter) Error(msg string) {
	styles := w.getStyles()
	_, _ = fmt.Fprintln(w.getErr(), styles.Error.Render(msg))
}
