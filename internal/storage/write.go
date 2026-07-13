package storage

import (
	"os"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func (s *Store) WriteItem(it *datamodel.Item) (string, error) {
	return s.WriteItemRaw(it.ID, codec.Serialize(it))
}

func (s *Store) WriteItemRaw(ulid, content string) (string, error) {
	if err := os.MkdirAll(s.ItemsDir(), 0o755); err != nil {
		return "", errx.User("creating tickets dir: %v", err)
	}
	dst := s.ItemPath(ulid)
	tmp := filepath.Join(s.ItemsDir(), "."+ulid+".md.tmp")
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return "", errx.User("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", errx.User("writing temp file: %v", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", errx.User("syncing temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", errx.User("closing temp file: %v", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return "", errx.User("renaming into place: %v", err)
	}
	return s.RelToRoot(dst), nil
}
