package gitx

import "os/exec"

func (r Repo) ShowCmd(sha string) *exec.Cmd {
	return gitCommand(r.Dir, nil, "show", sha)
}
