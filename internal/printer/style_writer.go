package printer

import (
	"fmt"
	"os"
	"test-cli/internal/tui"
)

type Writer interface {
	Info(msg string) error
	Success(msg string) error
	Error(msg string) error
}

type StyleWriter struct {
	styles tui.Styles
}

func NewStyleWriter(styles tui.Styles) Writer {
	return &StyleWriter{styles: styles}
}

func (w StyleWriter) Info(msg string) error {
	_, err := fmt.Fprintln(os.Stderr, w.styles.Info.Render(msg))
	if err != nil {
		return err
	}
	return nil
}

func (w StyleWriter) Success(msg string) error {
	_, err := fmt.Fprintln(os.Stderr, w.styles.Success.Render(msg))
	if err != nil {
		return err
	}
	return nil
}

func (w StyleWriter) Error(msg string) error {
	// TODO: Add a check for if the message starts with Error/error and add it if not
	_, err := fmt.Fprintln(os.Stderr, w.styles.Error.Render(msg))
	if err != nil {
		return err
	}
	return nil
}
