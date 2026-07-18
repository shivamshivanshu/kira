package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestCompletePathSpawnsNoGit(t *testing.T) {
	bin := testutil.KiraBinary(t)
	dir := t.TempDir()
	env := append(testutil.HermeticEnvironment(), "PATH="+pathEnv())

	mustRun(t, dir, env, "git", "init")
	mustRun(t, dir, env, "git", "config", "user.email", "test@example.com")
	mustRun(t, dir, env, "git", "config", "user.name", "tester")
	mustRun(t, dir, env, bin, "init", "--key", "KIRA")
	mustRun(t, dir, env, bin, "create", "ticket", "--title", "Fix orderbook race", "--no-edit")
	mustRun(t, dir, env, bin, "index")

	shimDir, counter := testutil.GitCountingShim(t)
	cmd := exec.Command(bin, "__complete", "show", "")
	cmd.Dir = dir
	cmd.Env = append(testutil.HermeticEnvironment(),
		"PATH="+shimDir+string(os.PathListSeparator)+pathEnv(),
	)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("__complete: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "KIRA-1") {
		t.Fatalf("completion produced no suggestions from the cache:\n%s", out.String())
	}
	if n := testutil.CountSpawns(t, counter); n != 0 {
		t.Fatalf("completion path spawned git %d time(s); the invariant forbids any", n)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "index.lock")); !os.IsNotExist(err) {
		t.Fatalf("completion path left a .git/index.lock behind")
	}
}
