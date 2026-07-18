package e2e

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestTUICrashExitsFour(t *testing.T) {
	bin := testutil.KiraBinary(t)

	dir := t.TempDir()
	env := append(testutil.HermeticEnvironment(), "PATH="+pathEnv())
	mustRun(t, dir, env, "git", "init")
	mustRun(t, dir, env, "git", "config", "user.email", "t@e.com")
	mustRun(t, dir, env, "git", "config", "user.name", "t")
	mustRun(t, dir, env, bin, "init", "--key", "KIRA")

	crash := exec.Command(bin, "tui", "--test-inject-panic")
	crash.Dir = dir
	crash.Env = env
	err := crash.Run()

	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected non-zero exit, got %v", err)
	}
	if ee.ExitCode() != 4 {
		t.Fatalf("exit code = %d, want 4", ee.ExitCode())
	}
}

func mustRun(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func pathEnv() string {
	if p, err := exec.LookPath("git"); err == nil {
		return filepath.Dir(p)
	}
	return "/usr/bin:/bin"
}
