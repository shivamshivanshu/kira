package fzfx

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func exitErr(t *testing.T, code int) error {
	t.Helper()
	err := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code)).Run()
	if err == nil {
		t.Fatalf("exit %d produced no error", code)
	}
	return err
}

func TestClassifyCancel(t *testing.T) {
	t.Parallel()
	for _, code := range []int{1, 130} {
		if err := classify(exitErr(t, code)); !errors.Is(err, ErrCancelled) {
			t.Errorf("classify(exit %d) = %v, want ErrCancelled", code, err)
		}
	}
}

func TestClassifyFailure(t *testing.T) {
	t.Parallel()
	for _, code := range []int{2, 3} {
		err := classify(exitErr(t, code))
		if errors.Is(err, ErrCancelled) {
			t.Fatalf("classify(exit %d) = ErrCancelled, want a failure", code)
		}
		if !strings.Contains(err.Error(), fmt.Sprintf("%d", code)) {
			t.Errorf("classify(exit %d) = %q, want the exit code in the message", code, err)
		}
	}
}

func TestClassifyNonExit(t *testing.T) {
	t.Parallel()
	cause := errors.New("spawn failed")
	err := classify(cause)
	if !errors.Is(err, cause) {
		t.Errorf("classify(%v) = %v, want it wrapped", cause, err)
	}
	if errors.Is(err, ErrCancelled) {
		t.Errorf("classify(%v) = ErrCancelled, want a failure", cause)
	}
}
