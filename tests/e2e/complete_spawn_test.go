package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletePathSpawnsNoGit(t *testing.T) {
	bin := kiraBinary(t)
	dir := t.TempDir()
	env := []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "EDITOR=true", "PATH=" + pathEnv()}

	mustRun(t, dir, env, "git", "init")
	mustRun(t, dir, env, "git", "config", "user.email", "test@example.com")
	mustRun(t, dir, env, "git", "config", "user.name", "tester")
	mustRun(t, dir, env, bin, "init", "--key", "KIRA")
	mustRun(t, dir, env, bin, "create", "ticket", "--title", "Fix orderbook race", "--no-edit")
	mustRun(t, dir, env, bin, "index")

	shimDir, counter := gitCountingShim(t)
	cmd := exec.Command(bin, "__complete", "show", "")
	cmd.Dir = dir
	cmd.Env = []string{
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "EDITOR=true",
		"PATH=" + shimDir + string(os.PathListSeparator) + pathEnv(),
		"KIRA_SPAWN_COUNTER=" + counter,
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("__complete: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "KIRA-1") {
		t.Fatalf("completion produced no suggestions from the cache:\n%s", out.String())
	}
	if n := spawnLines(t, counter); n != 0 {
		t.Fatalf("completion path spawned git %d time(s); the invariant forbids any", n)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "index.lock")); !os.IsNotExist(err) {
		t.Fatalf("completion path left a .git/index.lock behind")
	}
}

func kiraBinary(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "kira")
	if err := os.Symlink(self, bin); err != nil {
		t.Fatal(err)
	}
	return bin
}

func gitCountingShim(t *testing.T) (dir, counter string) {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("locate git: %v", err)
	}
	dir = t.TempDir()
	counter = filepath.Join(dir, "count")
	script := "#!/bin/sh\nprintf 'x\\n' >> \"$KIRA_SPAWN_COUNTER\"\nexec " + gitPath + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, counter
}

func spawnLines(t *testing.T, counter string) int {
	t.Helper()
	data, err := os.ReadFile(counter)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatal(err)
	}
	return bytes.Count(data, []byte{'\n'})
}
