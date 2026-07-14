package fzfx

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Options struct {
	Prompt     string
	PreviewCmd string
}

var ErrCancelled = errors.New("fzf: cancelled")

func Installed() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

func Pick(rows []string, opts Options) (string, error) {
	var args []string
	if opts.Prompt != "" {
		args = append(args, "--prompt", opts.Prompt)
	}
	if opts.PreviewCmd != "" {
		args = append(args, "--preview", opts.PreviewCmd)
	}
	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(rows, "\n") + "\n")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", classify(err)
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

func classify(err error) error {
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return fmt.Errorf("running fzf: %w", err)
	}
	switch ee.ExitCode() {
	case 1, 130:
		return ErrCancelled
	}
	return fmt.Errorf("fzf failed with exit code %d", ee.ExitCode())
}
