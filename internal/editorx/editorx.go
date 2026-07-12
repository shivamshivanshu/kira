package editorx

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func Command() ([]string, error) {
	ed := strings.TrimSpace(os.Getenv("EDITOR"))
	if ed == "" {
		ed = strings.TrimSpace(os.Getenv("VISUAL"))
	}
	if ed == "" {
		return nil, errors.New("$EDITOR is not set")
	}
	return strings.Fields(ed), nil
}

func Edit(path string) error {
	editor, err := Command()
	if err != nil {
		return err
	}
	args := append(slices.Clone(editor[1:]), path)
	cmd := exec.Command(editor[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %s: %v", filepath.Base(editor[0]), err)
	}
	return nil
}
