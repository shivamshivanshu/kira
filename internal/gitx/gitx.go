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
)

type Repo struct{ Dir string }

func Installed() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func (r Repo) Output(args ...string) (string, error) {
	out, err := r.OutputRaw(args...)
	return strings.TrimSpace(out), err
}

func (r Repo) OutputRaw(args ...string) (string, error) {
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
		return "", &CmdError{msg: fmt.Sprintf("git %s: %s", strings.Join(args, " "), msg)}
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

func (r Repo) logPatch(args ...string) (string, error) {
	format := "--format=" + nulFmt + "%H" + nulFmt + "%cI"
	return r.Output(append([]string{"log", format, "-p", fullFileContext}, args...)...)
}

func (r Repo) FileLog(relPath string) (string, error) {
	return r.logPatch("--", relPath)
}

func (r Repo) RangePatch(revRange, pathspec string) (string, error) {
	return r.logPatch(revRange, "--", pathspec)
}

func (r Repo) LastCommitFor(relPath string) (string, error) {
	return r.Output("log", "-1", "--format=%H", "--", relPath)
}

func (r Repo) LastCommits(pathspec string) (map[string]string, error) {
	out, err := r.Output("log", "--format="+nulFmt+"%H", "--name-only", "--", pathspec)
	if err != nil {
		return nil, err
	}
	shaByPath := map[string]string{}
	var sha string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, nul) {
			sha = line[len(nul):]
			continue
		}
		if line == "" {
			continue
		}
		if _, seen := shaByPath[line]; !seen {
			shaByPath[line] = sha
		}
	}
	return shaByPath, nil
}

func (r Repo) FileCommitMeta(relPath string) (string, error) {
	return r.Output("log", "--format=%H%x00%an%x00%cI%x00%P", "--", relPath)
}

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

func (r Repo) UnmergedPaths() ([]string, error) {
	return r.splitLines("diff", "--name-only", "--diff-filter=U")
}

func (r Repo) SetConfig(key, val string) error {
	_, err := r.Output("config", key, val)
	return err
}

type BatchObject struct {
	Content string
	Found   bool
}

func (r Repo) CatFileBatch(specs []string) ([]BatchObject, error) {
	cmd := exec.Command("git", "cat-file", "--batch")
	cmd.Dir = r.Dir
	cmd.Stdin = strings.NewReader(strings.Join(specs, "\n") + "\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git cat-file --batch: %s", strings.TrimSpace(stderr.String()))
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
		if pos+size > len(buf) {
			return nil, fmt.Errorf("git cat-file --batch: truncated content")
		}
		out = append(out, BatchObject{Content: string(buf[pos : pos+size]), Found: true})
		pos += size + 1
	}
	return out, nil
}

func MergeText(base, ours, theirs string) (merged string, conflict bool, err error) {
	dir, err := os.MkdirTemp("", "kira-merge")
	if err != nil {
		return "", false, err
	}
	defer os.RemoveAll(dir)

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

	cmd := exec.Command("git", "merge-file", "-p", op, bp, tp)
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
