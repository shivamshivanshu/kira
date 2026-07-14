package gitx

import (
	"os"
	"path/filepath"
	"strings"
)

func (r Repo) CurrentBranch() (string, error) {
	return r.Output("rev-parse", "--abbrev-ref", "HEAD")
}

func (r Repo) HeadBranchFast() (string, bool) {
	gitPath := filepath.Join(r.Dir, ".git")
	fi, err := os.Stat(gitPath)
	if err != nil {
		return "", false
	}
	if !fi.IsDir() {
		data, err := os.ReadFile(gitPath)
		if err != nil {
			return "", false
		}
		dir := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(data)), "gitdir:"))
		if dir == "" {
			return "", false
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(r.Dir, dir)
		}
		gitPath = dir
	}
	head, err := os.ReadFile(filepath.Join(gitPath, "HEAD"))
	if err != nil {
		return "", false
	}
	ref, ok := strings.CutPrefix(strings.TrimSpace(string(head)), "ref: refs/heads/")
	if !ok || ref == "" {
		return "", false
	}
	return ref, true
}

func (r Repo) Branches() ([]string, error) {
	return r.splitLines("for-each-ref", "--format=%(refname:short)", "refs/heads")
}

func (r Repo) Checkout(branch string) error {
	_, err := r.Output("checkout", branch)
	return err
}

func (r Repo) CheckoutNew(branch string) error {
	_, err := r.Output("checkout", "-b", branch)
	return err
}

func (r Repo) WorktreeAdd(path, branch string, create bool) error {
	args := []string{"worktree", "add"}
	if create {
		args = append(args, "-b", branch, path)
	} else {
		args = append(args, path, branch)
	}
	_, err := r.Output(args...)
	return err
}

func (r Repo) WorktreeForBranch(branch string) (string, bool) {
	out, err := r.Output("worktree", "list", "--porcelain")
	if err != nil {
		return "", false
	}
	var path string
	for _, line := range strings.Split(out, "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok {
			path = p
		} else if line == "branch refs/heads/"+branch {
			return path, true
		}
	}
	return "", false
}
