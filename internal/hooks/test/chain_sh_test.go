package hooks_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/hooks"
)

// (exit N), not a bare exit N: a bare exit would return before ever
// reaching the chained tail appended after it in the same script.
const (
	userHookSucceeds = "#!/bin/sh\ntrue\n"
	userHookFails    = "#!/bin/sh\n(exit 7)\n"
)

func runChainedHook(t *testing.T, dir, userScript, shimScript string) int {
	t.Helper()
	hookPath := filepath.Join(dir, "hook")
	chained := hooks.Chain(userScript, hooks.PostMerge)
	if err := os.WriteFile(hookPath, []byte(chained), 0o755); err != nil {
		t.Fatalf("write hook: %v", err)
	}
	if shimScript != "" {
		shimDir := filepath.Join(dir, ".kira", "hooks")
		if err := os.MkdirAll(shimDir, 0o755); err != nil {
			t.Fatalf("mkdir shim dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(shimDir, hooks.PostMerge), []byte(shimScript), 0o755); err != nil {
			t.Fatalf("write shim: %v", err)
		}
	}
	cmd := exec.Command("sh", hookPath)
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("run chained hook: %v", err)
	}
	return exitErr.ExitCode()
}

func TestChainedHookMissingShimExitsZero(t *testing.T) {
	code := runChainedHook(t, t.TempDir(), userHookSucceeds, "")
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (a checkout predating kira must not abort the chain)", code)
	}
}

func TestChainedHookFailingUserHookExitsNonzero(t *testing.T) {
	shim := "#!/bin/sh\nexit 0\n"
	code := runChainedHook(t, t.TempDir(), userHookFails, shim)
	if code == 0 {
		t.Error("exit code = 0, want nonzero: a failing user hook must not be masked by the shim's success")
	}
}

func TestChainedHookShimInvokedExactlyOnce(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "invoked")
	shim := "#!/bin/sh\necho x >> " + marker + "\nexit 0\n"
	if code := runChainedHook(t, dir, userHookSucceeds, shim); code != 0 {
		t.Fatalf("run chained hook: exit %d", code)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	if got := len(data); got != len("x\n") {
		t.Errorf("shim invoked %d times, want exactly 1 (marker: %q)", got/len("x\n"), string(data))
	}
}

func TestChainedHookSucceedingUserHookAndShimExitsZero(t *testing.T) {
	shim := "#!/bin/sh\nexit 0\n"
	code := runChainedHook(t, t.TempDir(), userHookSucceeds, shim)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestChainedHookFailingShimExitsNonzero(t *testing.T) {
	shim := "#!/bin/sh\nexit 9\n"
	code := runChainedHook(t, t.TempDir(), userHookSucceeds, shim)
	if code == 0 {
		t.Error("exit code = 0, want nonzero: a failing shim must fail the chain")
	}
}
