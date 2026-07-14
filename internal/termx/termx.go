package termx

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
)

func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func WriterIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && IsTerminal(f)
}

func Width(f *os.File) (int, bool) {
	w, _, err := term.GetSize(f.Fd())
	if err != nil || w <= 0 {
		return 0, false
	}
	return w, true
}

func IsInteractive() bool {
	return IsTerminal(os.Stdin) && IsTerminal(os.Stderr)
}

func Confirm(prompt string) bool {
	switch strings.ToLower(ReadLineDefault(prompt, "")) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func ReadLineDefault(prompt, def string) string {
	fmt.Fprint(os.Stderr, prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return def
	}
	if v := strings.TrimSpace(line); v != "" {
		return v
	}
	return def
}
