package gitx

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Repo struct{ Dir string }

func Installed() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func (r Repo) Output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r Repo) InsideWorkTree() error {
	_, err := r.Output("rev-parse", "--is-inside-work-tree")
	return err
}

func (r Repo) Stage(paths ...string) error {
	_, err := r.Output(append([]string{"add", "--"}, paths...)...)
	return err
}

func (r Repo) Commit(subject, trailerKey, trailerVal string) error {
	args := []string{"commit", "-m", subject}
	if trailerVal != "" {
		args = append(args, "-m", trailerKey+": "+trailerVal)
	}
	_, err := r.Output(args...)
	return err
}

func (r Repo) FollowLogPatch(relPath string) (string, error) {
	return r.Output("log", "--follow", "--format=%x01%cI", "-p", "--", relPath)
}
