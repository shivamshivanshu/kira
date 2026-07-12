package core

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// isInteractive reports whether stdin and stderr are both attached to a
// terminal (a character device), the precondition for prompting (project key,
// commit y/n). Detected via the file mode to avoid a terminal-detection
// dependency.
func isInteractive() bool {
	return IsTerminal(os.Stdin) && IsTerminal(os.Stderr)
}

// IsTerminal reports whether f is attached to a terminal (a character device).
// Detected via the file mode to avoid a terminal-detection dependency; shared
// by core's prompting and cli's interactive-picker gating.
func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// confirm asks a yes/no question on stderr (keeping stdout clean for --json)
// and reads the answer from stdin. A bare Enter or anything but y/yes is "no".
func confirm(action string) bool {
	fmt.Fprintf(os.Stderr, "commit %q? [y/N] ", action)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

// promptKey asks for the project key on stderr, offering def as the default.
func promptKey(def string) string {
	fmt.Fprintf(os.Stderr, "project key [%s]: ", def)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return def
	}
	if v := strings.TrimSpace(line); v != "" {
		return v
	}
	return def
}
