package tui

import (
	"fmt"
	"strings"
)

// progressBar renders a visual progress bar with the given fill (0.0 to 1.0).
func progressBar(f float64) string {
	blocks := 10
	filled := int(f * float64(blocks))
	if filled > blocks {
		filled = blocks
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", blocks-filled)
}

// TerminalReporter renders progress to the terminal using a spinner.
type TerminalReporter struct {
	Spinner *Spinner
}

// Report implements ProgressReporter by updating the spinner with progress info.
func (r *TerminalReporter) Report(projectID string, progress float64, message string) {
	if r.Spinner == nil {
		return
	}

	if progress >= 1.0 {
		r.Spinner.Stop()
		return
	}

	pct := int(progress * 100)
	msg := fmt.Sprintf("%s: [%s] %d%% %s", projectID, progressBar(progress), pct, message)
	r.Spinner.Update(msg)
}
