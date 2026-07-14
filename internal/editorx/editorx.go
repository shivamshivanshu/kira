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

func Command(configured string) (string, error) {
	for _, candidate := range []string{configured, os.Getenv("EDITOR"), os.Getenv("VISUAL")} {
		if ed := strings.TrimSpace(candidate); ed != "" {
			return ed, nil
		}
	}
	return "", errors.New("no editor configured: set ui.editor or $EDITOR")
}

func shellCommand(editor string, args ...string) *exec.Cmd {
	return exec.Command("sh", append([]string{"-c", editor + ` "$@"`, editor}, args...)...)
}

func editorName(editor string) string {
	return filepath.Base(strings.Fields(editor)[0])
}

func Edit(configured, path string, stdio Stdio) error {
	editor, err := Command(configured)
	if err != nil {
		return err
	}
	stdio = stdio.withDefaults()
	cmd := shellCommand(editor, path)
	cmd.Stdin = stdio.In
	cmd.Stdout = stdio.Out
	cmd.Stderr = stdio.Err
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %s: %v", editorName(editor), err)
	}
	return nil
}

var vimFamily = map[string]bool{"vi": true, "vim": true, "nvim": true, "gvim": true}

func View(configured, path string) (*exec.Cmd, error) {
	editor, err := Command(configured)
	if err != nil {
		return nil, err
	}
	args := []string{path}
	if vimFamily[editorName(editor)] {
		args = []string{"-R", path}
	}
	return shellCommand(editor, args...), nil
}
