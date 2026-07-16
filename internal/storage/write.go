package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

// WriteItem serializes it and writes it to its item file, returning the path
// relative to the store root.
func (s *FS) WriteItem(it *datamodel.Item) (string, error) {
	it.Body = codec.CanonicalizeCommentBody(it.Body)
	return s.WriteItemRaw(it.ID, codec.Serialize(it))
}

// WriteItemRaw writes content to the item file for ulid, returning the path
// relative to the store root.
func (s *FS) WriteItemRaw(ulid, content string) (string, error) {
	if err := os.MkdirAll(s.ItemsDir(), 0o755); err != nil {
		return "", errx.User("creating tickets dir: %v", err)
	}
	dst := s.ItemPath(ulid)
	if err := WriteFileAtomic(dst, []byte(content)); err != nil {
		return "", errx.User("writing ticket file: %v", err)
	}
	return s.RelToRoot(dst), nil
}

// WriteFileAtomic writes content to dst via a hidden temp file in the same
// directory, syncing both the file and the directory so the rename survives
// a crash.
func WriteFileAtomic(dst string, content []byte) error {
	dir := filepath.Dir(dst)
	tmp := filepath.Join(dir, "."+filepath.Base(dst)+".tmp")
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()

	if _, err := f.Write(content); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		return err
	}
	committed = true
	return syncDir(dir)
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	err = d.Sync()
	if err != nil && runtime.GOOS == "darwin" && errors.Is(err, syscall.EINVAL) {
		return nil
	}
	return err
}
