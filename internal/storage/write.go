package storage

import (
	"os"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func (s *FS) WriteItem(it *datamodel.Item) (string, error) {
	it.Body = codec.CanonicalizeCommentBody(it.Body)
	return s.WriteItemRaw(it.ID, codec.Serialize(it))
}

func (s *FS) WriteItemRaw(ulid, content string) (string, error) {
	if err := os.MkdirAll(s.ItemsDir(), 0o755); err != nil {
		return "", errx.User("creating tickets dir: %v", err)
	}
	dst := s.ItemPath(ulid)
	af, err := createAtomicFile(s.ItemsDir(), "."+ulid+itemExt+".tmp")
	if err != nil {
		return "", errx.User("creating temp file: %v", err)
	}
	defer af.cleanup()

	if err := af.write(content); err != nil {
		return "", errx.User("writing temp file: %v", err)
	}
	if err := af.sync(); err != nil {
		return "", errx.User("syncing temp file: %v", err)
	}
	if err := af.close(); err != nil {
		return "", errx.User("closing temp file: %v", err)
	}
	if err := af.commit(dst); err != nil {
		return "", errx.User("renaming into place: %v", err)
	}
	return s.RelToRoot(dst), nil
}

type atomicFile struct {
	f         *os.File
	path      string
	closed    bool
	committed bool
}

func createAtomicFile(dir, name string) (*atomicFile, error) {
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	return &atomicFile{f: f, path: path}, nil
}

func (a *atomicFile) write(content string) error {
	_, err := a.f.WriteString(content)
	return err
}

func (a *atomicFile) sync() error { return a.f.Sync() }

func (a *atomicFile) close() error {
	if a.closed {
		return nil
	}
	a.closed = true
	return a.f.Close()
}

func (a *atomicFile) commit(dst string) error {
	if err := os.Rename(a.path, dst); err != nil {
		return err
	}
	a.committed = true
	return nil
}

func (a *atomicFile) cleanup() {
	if a.committed {
		return
	}
	a.close()
	os.Remove(a.path)
}
