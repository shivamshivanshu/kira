// Package gitx wraps the git CLI: command execution, index and staging, branches, trailers, and sync.
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

	"github.com/shivamshivanshu/kira/internal/setx"
)

// shortSHALen matches git's own default abbreviation length.
const shortSHALen = 7

// ShortSHA truncates a commit SHA to git's default short form.
func ShortSHA(sha string) string {
	if len(sha) > shortSHALen {
		return sha[:shortSHALen]
	}
	return sha
}

// Repo is a git working tree rooted at Dir.
type Repo struct{ Dir string }

// Installed reports whether a git executable is on PATH.
func Installed() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Output runs git with args in r.Dir and returns trimmed stdout.
func (r Repo) Output(args ...string) (string, error) {
	out, err := r.OutputRaw(args...)
	return strings.TrimSpace(out), err
}

func gitCommand(dir string, env []string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(), "LC_ALL=C", "LANG=C"), env...)
	return cmd
}

// OutputRaw runs git with args in r.Dir and returns stdout untrimmed.
func (r Repo) OutputRaw(args ...string) (string, error) {
	return r.outputRaw(nil, args...)
}

func (r Repo) outputRaw(env []string, args ...string) (string, error) {
	cmd := gitCommand(r.Dir, env, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", cmdError("git "+strings.Join(args, " "), &stderr, err)
	}
	return stdout.String(), nil
}

func (r Repo) splitLines(args ...string) ([]string, error) {
	out, err := r.Output(args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// InsideWorkTree returns an error unless r.Dir is inside a git work tree.
func (r Repo) InsideWorkTree() error {
	_, err := r.Output("rev-parse", "--is-inside-work-tree")
	return err
}

// Stage runs `git add` on paths.
func (r Repo) Stage(paths ...string) error {
	_, err := r.Output(append([]string{"add", "--"}, paths...)...)
	return err
}

// Unstage runs `git restore --staged` on paths.
func (r Repo) Unstage(paths ...string) error {
	_, err := r.Output(append([]string{"restore", "--staged", "--"}, paths...)...)
	return err
}

// Commit runs `git commit` with subject as the first -m and, if trailerVal
// is set, a second -m of "trailerKey: trailerVal".
func (r Repo) Commit(subject, trailerKey, trailerVal string) error {
	if trailerVal == "" {
		return r.CommitParts(subject)
	}
	return r.CommitParts(subject, trailerKey+": "+trailerVal)
}

// CommitParts runs `git commit` with each part as its own -m.
func (r Repo) CommitParts(parts ...string) error {
	return r.CommitScoped(nil, parts...)
}

// CommitScoped runs `git commit` with each part as its own -m, scoped to
// pathspecs if any are given.
func (r Repo) CommitScoped(pathspecs []string, parts ...string) error {
	args := []string{"commit"}
	for _, p := range parts {
		args = append(args, "-m", p)
	}
	if len(pathspecs) > 0 {
		args = append(append(args, "--"), pathspecs...)
	}
	_, err := r.Output(args...)
	return err
}

const fullFileContext = "--unified=1000000"

func (r Repo) logPatch(args ...string) (string, error) {
	format := "--format=" + nulFmt + "%H" + nulFmt + "%cI"
	return r.Output(append([]string{"log", format, "-p", fullFileContext}, args...)...)
}

// FileLog returns the full-context patch log for relPath.
func (r Repo) FileLog(relPath string) (string, error) {
	return r.logPatch("--", relPath)
}

// RangePatch returns the full-context patch log for pathspec over revRange.
func (r Repo) RangePatch(revRange, pathspec string) (string, error) {
	return r.logPatch(revRange, "--", pathspec)
}

// LastCommitFor returns the SHA of the most recent commit touching relPath.
func (r Repo) LastCommitFor(relPath string) (string, error) {
	return r.Output("log", "-1", "--format=%H", "--", relPath)
}

// LastCommits returns, for every path under pathspec, the SHA of its most
// recent commit, keyed by path relative to r.Dir.
func (r Repo) LastCommits(pathspec string) (map[string]string, error) {
	out, err := r.Output("log", "--format="+nulFmt+"%H", "--name-only", "--", pathspec)
	if err != nil {
		return nil, err
	}
	var paths, shas []string
	dedup := setx.NewDeduper[string]()
	var sha string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, nul) {
			sha = line[len(nul):]
			continue
		}
		if line == "" || !dedup.Add(line) {
			continue
		}
		paths = append(paths, line)
		shas = append(shas, sha)
	}
	rel, err := r.relToDir(paths)
	if err != nil {
		return nil, err
	}
	shaByPath := make(map[string]string, len(rel))
	for i, p := range rel {
		shaByPath[p] = shas[i]
	}
	return shaByPath, nil
}

// FileCommitMeta returns SHA, author, commit date, and parents for every
// commit touching relPath.
func (r Repo) FileCommitMeta(relPath string) (string, error) {
	return r.Output("log", "--format=%H%x00%an%x00%cI%x00%P", "--", relPath)
}

// RevListSince returns the commits on rev since the given duration/date
// expression.
func (r Repo) RevListSince(rev, since string) ([]Commit, error) {
	format := "--format=%H" + nulFmt + "%s" + nulFmt + "%an" + nulFmt + "%cI"
	out, err := r.Output("rev-list", "--since="+since, format, rev)
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, nul)
		if len(f) != 4 {
			continue
		}
		commits = append(commits, Commit{SHA: f[0], Subject: f[1], Author: f[2], Timestamp: f[3]})
	}
	return commits, nil
}

// GitPath resolves rel to an absolute path under the repo's .git directory.
func (r Repo) GitPath(rel string) (string, error) {
	out, err := r.Output("rev-parse", "--git-path", rel)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(r.Dir, out)
	}
	return out, nil
}

// AppendInfoAttribute adds line to the repo's info/attributes file if it is
// not already present.
func (r Repo) AppendInfoAttribute(line string) error {
	path, err := r.GitPath(infoAttributesPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return AppendLineIfMissing(path, line)
}

func containsLine(content, line string) bool {
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == line {
			return true
		}
	}
	return false
}

// AppendLineIfMissing appends line to the file at path unless it already
// contains that line, creating the file's content fresh if it does not
// exist.
func AppendLineIfMissing(path, line string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if containsLine(string(existing), line) {
		return nil
	}
	body := string(existing)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body+line+"\n"), 0o644)
}

// RemoveLineIfPresent removes line from the file at path if present; it is a
// no-op if path does not exist.
func RemoveLineIfPresent(path, line string) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if !containsLine(string(existing), line) {
		return nil
	}
	kept := make([]string, 0)
	for l := range strings.Lines(string(existing)) {
		if strings.TrimSpace(l) == line {
			continue
		}
		kept = append(kept, l)
	}
	return os.WriteFile(path, []byte(strings.Join(kept, "")), 0o644)
}

// RebaseInProgress reports whether r is in the middle of a rebase.
func (r Repo) RebaseInProgress() bool {
	for _, rel := range []string{"rebase-merge", "rebase-apply"} {
		p, err := r.GitPath(rel)
		if err != nil {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// UnmergedPaths returns paths still marked unmerged (diff-filter=U).
func (r Repo) UnmergedPaths() ([]string, error) {
	lines, err := r.splitLines("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	return r.relToDir(lines)
}

// SetConfig runs `git config key val`.
func (r Repo) SetConfig(key, val string) error {
	_, err := r.Output("config", key, val)
	return err
}

// ConfigValue returns the value of key, or "" if unset.
func (r Repo) ConfigValue(key string) string {
	v, err := r.Output("config", "--get", key)
	if err != nil {
		return ""
	}
	return v
}

// UnsetConfig removes key if it is set.
func (r Repo) UnsetConfig(key string) error {
	if r.ConfigValue(key) == "" {
		return nil
	}
	_, err := r.Output("config", "--unset", key)
	return err
}

// BatchObject is one result from CatFileBatch.
type BatchObject struct {
	Content string
	Found   bool
}

// CatFileBatch runs `git cat-file --batch` over specs and returns one
// BatchObject per spec, in order.
func (r Repo) CatFileBatch(specs []string) ([]BatchObject, error) {
	cmd := gitCommand(r.Dir, nil, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(strings.Join(specs, "\n") + "\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, cmdError("git cat-file --batch", &stderr, err)
	}
	return parseCatFileBatch(stdout.Bytes(), len(specs))
}

func parseCatFileBatch(buf []byte, n int) ([]BatchObject, error) {
	out := make([]BatchObject, 0, n)
	pos := 0
	for range n {
		nl := bytes.IndexByte(buf[pos:], '\n')
		if nl < 0 {
			return nil, fmt.Errorf("git cat-file --batch: truncated header")
		}
		header := string(buf[pos : pos+nl])
		pos += nl + 1
		if strings.HasSuffix(header, " missing") {
			out = append(out, BatchObject{})
			continue
		}
		fields := strings.Fields(header)
		if len(fields) != 3 {
			return nil, fmt.Errorf("git cat-file --batch: bad header %q", header)
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("git cat-file --batch: bad size in %q", header)
		}
		if pos+size+1 > len(buf) {
			return nil, fmt.Errorf("git cat-file --batch: truncated content")
		}
		out = append(out, BatchObject{Content: string(buf[pos : pos+size]), Found: true})
		pos += size + 1
	}
	return out, nil
}

// MergeText runs a three-way `git merge-file` over base/ours/theirs in a
// scratch directory, reporting whether the result has conflict markers.
func MergeText(base, ours, theirs string) (merged string, conflict bool, err error) {
	dir, err := os.MkdirTemp("", "kira-merge")
	if err != nil {
		return "", false, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	write := func(name, content string) (string, error) {
		p := filepath.Join(dir, name)
		return p, os.WriteFile(p, []byte(content), 0o644)
	}
	op, err := write("ours", ours)
	if err != nil {
		return "", false, err
	}
	bp, err := write("base", base)
	if err != nil {
		return "", false, err
	}
	tp, err := write("theirs", theirs)
	if err != nil {
		return "", false, err
	}

	cmd := gitCommand("", nil, "merge-file", "-p", op, bp, tp)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr == nil {
		return stdout.String(), false, nil
	}
	var ee *exec.ExitError
	if errors.As(runErr, &ee) {
		if code := ee.ExitCode(); code > 0 && code < 128 {
			return stdout.String(), true, nil
		}
	}
	return "", false, fmt.Errorf("git merge-file: %s", strings.TrimSpace(stderr.String()))
}
