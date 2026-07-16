package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func (r Repo) ResolveTreeish(ref string) (string, error) {
	return r.Output("rev-parse", "--verify", "--quiet", ref+"^{commit}")
}

type Date string

type Ref string

func (r Repo) ResolveAtDate(date Date, anchor Ref) (string, error) {
	sha, err := r.Output("rev-list", "-1", "--before="+string(date), string(anchor))
	if err != nil {
		return "", err
	}
	if sha == "" {
		return "", fmt.Errorf("no commit on %s at or before %s", anchor, date)
	}
	return sha, nil
}

func (r Repo) MergeBase(a, b string) (string, error) {
	return r.Output("merge-base", a, b)
}

func (r Repo) LsTreeNames(treeish string, pathspecs ...string) ([]string, error) {
	args := append([]string{"ls-tree", "-r", "--name-only", treeish, "--"}, pathspecs...)
	return r.splitLines(args...)
}

func (r Repo) NumstatNoIndex(a, b string) (added, removed int, err error) {
	dir, err := os.MkdirTemp("", "kira-numstat")
	if err != nil {
		return 0, 0, err
	}
	defer os.RemoveAll(dir)
	ap := filepath.Join(dir, "a")
	bp := filepath.Join(dir, "b")
	if err := os.WriteFile(ap, []byte(a), 0o644); err != nil {
		return 0, 0, err
	}
	if err := os.WriteFile(bp, []byte(b), 0o644); err != nil {
		return 0, 0, err
	}
	cmd := gitCommand(r.Dir, nil, "diff", "--numstat", "--no-index", "--", ap, bp)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr != nil {
		var ee *exec.ExitError
		if !errors.As(runErr, &ee) || ee.ExitCode() >= 128 {
			return 0, 0, cmdError("git diff --numstat --no-index", &stderr, runErr)
		}
	}
	fields := strings.Fields(stdout.String())
	if len(fields) < 2 {
		return 0, 0, nil
	}
	added, _ = strconv.Atoi(fields[0])
	removed, _ = strconv.Atoi(fields[1])
	return added, removed, nil
}
