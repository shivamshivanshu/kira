package core

import (
	"os"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func (s *Store) ConfigSet(cfg *datamodel.Config, key, value string) (*datamodel.ConfigSetResult, error) {
	fs := s.fs()
	release, err := fs.Lock()
	if err != nil {
		return nil, err
	}
	defer release()

	data, err := os.ReadFile(fs.ConfigPath())
	if err != nil {
		return nil, errx.User("reading config: %v", err)
	}
	out, err := config.SetScalar(data, key, value)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	if err := os.WriteFile(fs.ConfigPath(), out, 0o644); err != nil {
		return nil, errx.User("writing config: %v", err)
	}
	subject := "kira: config set " + key
	if _, err := s.finalize(cfg.Commit.Mode, cfg.Commit.Trailer, subject, "", fs.RelToRoot(fs.ConfigPath())); err != nil {
		return nil, err
	}
	return &datamodel.ConfigSetResult{Key: key, Value: value}, nil
}
