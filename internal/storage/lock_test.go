//go:build unix

package storage

import (
	"errors"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/errx"
)

func withFastLockTimeout(t *testing.T) {
	t.Helper()
	origTimeout, origPoll := lockTimeout, lockPollInterval
	lockTimeout = 50 * time.Millisecond
	lockPollInterval = 5 * time.Millisecond
	t.Cleanup(func() {
		lockTimeout, lockPollInterval = origTimeout, origPoll
	})
}

func TestLockSecondCallWhileHeldReturnsConflict(t *testing.T) {
	withFastLockTimeout(t)
	s := New(t.TempDir())

	release, err := s.Lock()
	if err != nil {
		t.Fatalf("first Lock: %v", err)
	}
	defer release()

	_, err = s.Lock()
	if err == nil {
		t.Fatal("second Lock while held should fail")
	}
	var xerr *errx.Error
	if !errors.As(err, &xerr) || xerr.Code != errx.ExitConflict {
		t.Fatalf("want errx.Conflict, got %v", err)
	}
}

func TestLockAfterReleaseSucceeds(t *testing.T) {
	withFastLockTimeout(t)
	s := New(t.TempDir())

	release, err := s.Lock()
	if err != nil {
		t.Fatalf("first Lock: %v", err)
	}
	release()

	release2, err := s.Lock()
	if err != nil {
		t.Fatalf("Lock after release: %v", err)
	}
	release2()
}
