package core

import (
	"os"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/workon"
)

func (s *Store) activePath() string {
	return filepath.Join(s.fs().CacheDir(), "active")
}

func (s *Store) writeActive(p workon.ActivePointer) error {
	if err := os.MkdirAll(s.fs().CacheDir(), dirPerm); err != nil {
		return errx.Env("creating cache dir: %v", err)
	}
	if err := os.WriteFile(s.activePath(), p.Marshal(), filePerm); err != nil {
		return errx.Env("writing active pointer: %v", err)
	}
	return nil
}

func (s *Store) readActive() (workon.ActivePointer, bool) {
	data, err := os.ReadFile(s.activePath())
	if err != nil {
		return workon.ActivePointer{}, false
	}
	return workon.ParseActive(data)
}
