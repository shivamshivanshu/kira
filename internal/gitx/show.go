package gitx

import "os/exec"

func (r Repo) ShowCmd(sha string) *exec.Cmd {
	cmd := exec.Command("git", "show", sha)
	cmd.Dir = r.Dir
	return cmd
}
