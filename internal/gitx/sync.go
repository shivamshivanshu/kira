package gitx

func (r Repo) Pull(remote string) error {
	args := []string{"pull", "--rebase"}
	if remote != "" {
		args = append(args, remote)
	}
	_, err := r.Output(args...)
	return err
}

func (r Repo) Push(remote string) error {
	args := []string{"push"}
	if remote != "" {
		args = append(args, remote)
	}
	_, err := r.Output(args...)
	return err
}

func (r Repo) RebaseContinue() error {
	_, err := r.outputRaw([]string{"GIT_EDITOR=true"}, "rebase", "--continue")
	return err
}

func (r Repo) RebaseAbort() error {
	_, err := r.Output("rebase", "--abort")
	return err
}

func (r Repo) Stash() error {
	_, err := r.Output("stash", "push", "--include-untracked")
	return err
}

func (r Repo) StashPop() error {
	_, err := r.Output("stash", "pop")
	return err
}
