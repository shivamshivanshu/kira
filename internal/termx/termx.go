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
	return term.IsTerminal(f.Fd())
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

// stdinReader is shared across prompts: a per-call reader would buffer ahead
// and drop the remaining piped input between consecutive prompts.
var stdinReader *bufio.Reader

func ReadLineDefault(prompt, def string) string {
	fmt.Fprint(os.Stderr, prompt)
	if stdinReader == nil {
		stdinReader = bufio.NewReader(os.Stdin)
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil {
		return def
	}
	if v := strings.TrimSpace(line); v != "" {
		return v
	}
	return def
}
