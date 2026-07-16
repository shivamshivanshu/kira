package editorx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Stdio struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

func (s Stdio) withDefaults() Stdio {
	if s.In == nil {
		s.In = os.Stdin
	}
	if s.Out == nil {
		s.Out = os.Stdout
	}
	if s.Err == nil {
		s.Err = os.Stderr
	}
	return s
}

// ConfigHint is the remediation text for a "no editor configured" error.
const ConfigHint = "set ui.editor in config, or export $EDITOR/$VISUAL"

// Editor is only valid when constructed via Command; the zero value must not be used.
type Editor struct {
	raw string
}

func Command(configured string) (Editor, error) {
	for _, candidate := range []string{configured, os.Getenv("VISUAL"), os.Getenv("EDITOR")} {
		if ed := strings.TrimSpace(candidate); ed != "" {
			return Editor{raw: ed}, nil
		}
	}
	return Editor{}, errors.New("no editor configured")
}

func (e Editor) name() string {
	return filepath.Base(strings.Fields(e.raw)[0])
}

func (e Editor) build(args ...string) *exec.Cmd {
	return exec.Command("sh", append([]string{"-c", e.raw + ` "$@"`, e.raw}, args...)...)
}

func (e Editor) Edit(path string, stdio Stdio) error {
	stdio = stdio.withDefaults()
	cmd := e.build(path)
	cmd.Stdin = stdio.In
	cmd.Stdout = stdio.Out
	cmd.Stderr = stdio.Err
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %s: %w", e.name(), err)
	}
	return nil
}

var vimFamily = map[string]bool{"vi": true, "vim": true, "nvim": true, "gvim": true}

func (e Editor) View(path string) *exec.Cmd {
	args := []string{path}
	if vimFamily[e.name()] {
		args = []string{"-R", path}
	}
	return e.build(args...)
}
