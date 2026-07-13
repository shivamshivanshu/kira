package gitx

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func (r Repo) ToplevelHead() (toplevel, head string, err error) {
	if out, e := r.OutputRaw("rev-parse", "--is-inside-work-tree", "--show-toplevel", "HEAD"); e == nil {
		if lines := strings.Split(strings.TrimSpace(out), "\n"); len(lines) >= 3 && lines[0] == "true" {
			return lines[1], lines[2], nil
		}
	}
	out, e := r.OutputRaw("rev-parse", "--is-inside-work-tree", "--show-toplevel")
	if e != nil {
		return "", "", e
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 || lines[0] != "true" {
		return "", "", fmt.Errorf("not inside a git work tree")
	}
	return lines[1], "", nil
}

func (r Repo) IsAncestor(ancestor, descendant string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant)
	cmd.Dir = r.Dir
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func (r Repo) StatusPorcelain(pathspec string) ([]string, error) {
	out, err := r.OutputRaw("status", "--porcelain", "--", pathspec)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		rest := line[3:]
		if i := strings.Index(rest, " -> "); i >= 0 {
			rest = rest[i+len(" -> "):]
		}
		paths = append(paths, unquotePath(rest))
	}
	return paths, nil
}

func (r Repo) DiffNameStatus(from, to, pathspec string) ([]string, error) {
	out, err := r.Output("diff", "--name-status", from, to, "--", pathspec)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		paths = append(paths, unquotePath(fields[len(fields)-1]))
	}
	return paths, nil
}

func unquotePath(p string) string {
	if len(p) >= 2 && strings.HasPrefix(p, `"`) && strings.HasSuffix(p, `"`) {
		if unq, err := strconv.Unquote(p); err == nil {
			return unq
		}
	}
	return p
}
