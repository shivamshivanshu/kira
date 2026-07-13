package e2e

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBareTUINonTTYExitsThree(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "kira")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/kira").CombinedOutput(); err != nil {
		t.Fatalf("build kira: %v\n%s", err, out)
	}

	dir := t.TempDir()
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.Env = []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null", "PATH=" + pathEnv()}
	cmd.Stdin = strings.NewReader("")
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	err := cmd.Run()

	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected non-zero exit, got %v", err)
	}
	if ee.ExitCode() != 3 {
		t.Fatalf("exit code = %d, want 3 (non-tty uninitialized repo)", ee.ExitCode())
	}
}
