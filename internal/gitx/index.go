package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

type Ancestor string

type Descendant string

func (r Repo) IsAncestor(ancestor Ancestor, descendant Descendant) (bool, error) {
	cmd := gitCommand(r.Dir, "merge-base", "--is-ancestor", string(ancestor), string(descendant))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if exit.ExitCode() == 1 {
			return false, nil
		}
		if isUnknownRevision(stderr.String()) {
			return false, nil
		}
	}
	return false, &CmdError{msg: fmt.Sprintf("git merge-base --is-ancestor %s %s: %v", ancestor, descendant, err)}
}

func isUnknownRevision(stderr string) bool {
	s := strings.ToLower(stderr)
	return strings.Contains(s, "not a valid object name") ||
		strings.Contains(s, "not a valid commit name") ||
		strings.Contains(s, "bad revision")
}

var showPrefixCache sync.Map

func (r Repo) showPrefix() (string, error) {
	if v, ok := showPrefixCache.Load(r.Dir); ok {
		return v.(string), nil
	}
	prefix, err := r.Output("rev-parse", "--show-prefix")
	if err != nil {
		return "", err
	}
	showPrefixCache.Store(r.Dir, prefix)
	return prefix, nil
}

func (r Repo) relToDir(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return paths, nil
	}
	prefix, err := r.showPrefix()
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		return paths, nil
	}
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = relFromToplevel(prefix, p)
	}
	return out, nil
}

func relFromToplevel(prefix, p string) string {
	rel, err := filepath.Rel(filepath.FromSlash(prefix), filepath.FromSlash(p))
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}

func RevPath(rev, path string) string {
	return rev + ":./" + path
}

func parsePorcelainPaths(out string) []string {
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
	return paths
}

type DiffFrom string

type DiffTo string

func (r Repo) DiffNameStatus(from DiffFrom, to DiffTo, pathspec string) ([]string, error) {
	out, err := r.Output("diff", "--name-status", string(from), string(to), "--", pathspec)
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
	return r.relToDir(paths)
}

func unquotePath(p string) string {
	if len(p) >= 2 && strings.HasPrefix(p, `"`) && strings.HasSuffix(p, `"`) {
		if unq, err := strconv.Unquote(p); err == nil {
			return unq
		}
	}
	return p
}
