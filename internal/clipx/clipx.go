// Package clipx copies text to the terminal clipboard via OSC 52 (always
// emitted, tmux-passthrough-wrapped inside tmux) and, when a display is
// present, an external tool (pbcopy / wl-copy / xclip / xsel).
//
// Copy writes the OSC 52 sequence directly to Term. When Term is also driven
// by a concurrent renderer (bubbletea flushes frames from its own goroutine),
// the escape sequence can interleave with a frame; bubbletea v1 offers no API
// to emit raw sequences through the program, so callers accept this hazard
// until the program can route the emission itself.
package clipx

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Clipboard struct {
	Getenv   func(string) string
	GOOS     string
	LookPath func(string) (string, error)
	Exec     func(name string, args []string, stdin []byte) error
	Term     io.Writer
}

func System(term io.Writer) Clipboard {
	return Clipboard{
		Getenv:   os.Getenv,
		GOOS:     runtime.GOOS,
		LookPath: exec.LookPath,
		Exec:     runCmd,
		Term:     term,
	}
}

func (c Clipboard) Copy(text string) error {
	var errs []error
	if c.Term != nil {
		if _, err := io.WriteString(c.Term, OSC52(text, c.Getenv("TMUX") != "")); err != nil {
			errs = append(errs, fmt.Errorf("terminal: %w", err))
		}
	}
	if name, args := c.externalTool(); name != "" {
		if err := c.Exec(name, args, []byte(text)); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

func OSC52(text string, tmux bool) string {
	seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
	if tmux {
		return "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	return seq
}

func (c Clipboard) externalTool() (string, []string) {
	installed := func(name string) bool { _, err := c.LookPath(name); return err == nil }
	if c.GOOS == "darwin" {
		if installed("pbcopy") {
			return "pbcopy", nil
		}
		return "", nil
	}
	if c.Getenv("WAYLAND_DISPLAY") != "" && installed("wl-copy") {
		return "wl-copy", nil
	}
	if c.Getenv("DISPLAY") != "" {
		if installed("xclip") {
			return "xclip", []string{"-selection", "clipboard"}
		}
		if installed("xsel") {
			return "xsel", []string{"--clipboard", "--input"}
		}
	}
	return "", nil
}

func runCmd(name string, args []string, stdin []byte) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(stdin)
	return cmd.Run()
}
