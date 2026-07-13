package gitx

import "strings"

func (r Repo) CurrentBranch() (string, error) {
	return r.Output("rev-parse", "--abbrev-ref", "HEAD")
}

func (r Repo) Branches() ([]string, error) {
	return r.splitLines("for-each-ref", "--format=%(refname:short)", "refs/heads")
}

func (r Repo) HasBranch(name string) bool {
	_, err := r.Output("rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
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
