package gitx

import (
	"os"
	"os/exec"
	"testing"
)

func gitTry(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e.c",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e.c")
	return cmd.CombinedOutput()
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := gitTry(dir, args...); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}
