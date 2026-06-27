package printer

import (
	"fmt"
	"os"
	"sattchel/internal/tui"
)

type Writer interface {
	Info(msg string)
	Success(msg string)
	Error(msg string)
}

type StyleWriter struct {
	styles tui.Styles
}

func NewStyleWriter(styles tui.Styles) Writer {
	return &StyleWriter{styles: styles}
}

func (w StyleWriter) Info(msg string) {
	fmt.Fprintln(os.Stdout, w.styles.Info.Render(msg))
}

func (w StyleWriter) Success(msg string) {
	fmt.Fprintln(os.Stdout, w.styles.Success.Render(msg))
}

func (w StyleWriter) Error(msg string) {
	// TODO: Add a check for if the message starts with Error/error and add it if not
	fmt.Fprintln(os.Stderr, w.styles.Error.Render(msg))
}
