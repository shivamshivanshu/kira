package fzfx

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Options struct {
	Prompt     string
	PreviewCmd string
}

func Installed() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

func Pick(rows []string, opts Options) (string, bool, error) {
	var stdin strings.Builder
	for _, r := range rows {
		stdin.WriteString(r)
		stdin.WriteByte('\n')
	}
	args := []string{"--with-nth", "1..", "--prompt", opts.Prompt}
	if opts.PreviewCmd != "" {
		args = append(args, "--preview", opts.PreviewCmd)
	}
	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(stdin.String())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", true, nil
		}
		return "", false, fmt.Errorf("running fzf: %v", err)
	}
	return strings.TrimSuffix(string(out), "\n"), false, nil
}
