package e2e

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/testutil"
)

func TestBareTUINonTTYExitsThree(t *testing.T) {
	bin := testutil.KiraBinary(t)

	dir := t.TempDir()
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.Env = append(testutil.HermeticEnvironment(), "PATH="+pathEnv())
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
