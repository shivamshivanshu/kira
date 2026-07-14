package storage

import (
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/shivamshivanshu/kira/internal/errx"
)

const (
	lockTimeout      = 2 * time.Second
	lockPollInterval = 20 * time.Millisecond
)

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
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			return func() {
				syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
				f.Close()
			}, nil
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, errx.Conflict("another kira process holds the lock on %s", s.root)
		}
		time.Sleep(lockPollInterval)
	}
}
