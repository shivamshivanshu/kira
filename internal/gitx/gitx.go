package gitx

import (
	"os/exec"
	"strings"
)

type Repo struct{ Dir string }

func Installed() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func (r Repo) Output(args ...string) (string, error) {
	out, err := r.outputRaw(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
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

const fullFileContext = "--unified=1000000"

func (r Repo) FileLog(relPath string) (string, error) {
	return r.Output("log", "--format=%x00%H%x00%cI", "-p", fullFileContext, "--", relPath)
}

func (r Repo) LastCommitFor(relPath string) (string, error) {
	return r.Output("log", "-1", "--format=%H", "--", relPath)
}

func (r Repo) FileCommitMeta(relPath string) (string, error) {
	return r.Output("log", "--format=%H%x00%an%x00%cI%x00%P", "--", relPath)
}
