package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunPager pipes the formatted text content to the terminal pager (e.g. less -R).
func RunPager(content string) error {
	// If stdout is not a TTY (e.g. piped to another command), just print the raw content
	if stat, err := os.Stdout.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		fmt.Print(content)
		return nil
	}

	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}

	var cmd *exec.Cmd
	if pager == "less" {
		// -R: show ANSI color codes
		// -F: quit if the entire file fits on one screen
		// -X: do not clear screen on exit
		cmd = exec.Command("less", "-R", "-F", "-X")
	} else {
		args := strings.Fields(pager)
		if len(args) == 0 {
			fmt.Print(content)
			return nil
		}
		cmd = exec.Command(args[0], args[1:]...)
	}

	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
