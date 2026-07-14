package editorx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

func Command(configured string) ([]string, error) {
	for _, candidate := range []string{configured, os.Getenv("EDITOR"), os.Getenv("VISUAL")} {
		if ed := strings.TrimSpace(candidate); ed != "" {
			return strings.Fields(ed), nil
		}
	}
	return nil, errors.New("no editor configured: set ui.editor or $EDITOR")
}

func Edit(configured, path string, stdio Stdio) error {
	editor, err := Command(configured)
	if err != nil {
		return err
	}
	stdio = stdio.withDefaults()
	args := append(slices.Clone(editor[1:]), path)
	cmd := exec.Command(editor[0], args...)
	cmd.Stdin = stdio.In
	cmd.Stdout = stdio.Out
	cmd.Stderr = stdio.Err
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %s: %v", filepath.Base(editor[0]), err)
	}
	return nil
}

var vimFamily = map[string]bool{"vi": true, "vim": true, "nvim": true, "gvim": true}

func View(configured, path string) (*exec.Cmd, error) {
	editor, err := Command(configured)
	if err != nil {
		return nil, err
	}
	args := slices.Clone(editor[1:])
	if vimFamily[filepath.Base(editor[0])] {
		args = append(args, "-R")
	}
	return exec.Command(editor[0], append(args, path)...), nil
}
