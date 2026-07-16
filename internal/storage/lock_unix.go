//go:build unix

package storage

import (
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"github.com/shivamshivanshu/kira/internal/errx"
)

var (
	lockTimeout      = 2 * time.Second
	lockPollInterval = 20 * time.Millisecond
)

// Lock acquires an exclusive, cross-process lock on the store and returns a
// function that releases it.
func (s *FS) Lock() (func(), error) {
	if err := os.MkdirAll(s.CacheDir(), 0o755); err != nil {
		return nil, errx.User("creating cache dir: %v", err)
	}
	path := filepath.Join(s.CacheDir(), "lock")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, errx.User("opening lock: %v", err)
	}
	deadline := time.Now().Add(lockTimeout)
	for {
		if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err == nil {
			return func() {
				_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
				_ = f.Close()
			}, nil
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, errx.Conflict("another kira process holds the lock on %s", s.root)
		}
		time.Sleep(lockPollInterval)
	}
}
