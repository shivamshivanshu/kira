// Package index maintains the disposable SQLite cache under .kira/.cache/; it is
// never authoritative and is fully rebuildable from the ticket files plus git.
package index

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/shivamshivanshu/kira/internal/errx"
)

type Index struct {
	db       *sql.DB
	cacheDir string
}

func dbPath(cacheDir string) string   { return filepath.Join(cacheDir, "index.db") }
func metaPath(cacheDir string) string { return filepath.Join(cacheDir, "meta.json") }

func Open(cacheDir string) (*Index, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, errx.User("creating cache dir: %v", err)
	}
	idx, err := open(cacheDir)
	if err == nil {
		return idx, nil
	}
	discard(cacheDir)
	return open(cacheDir)
}

func open(cacheDir string) (*Index, error) {
	dsn := "file:" + dbPath(cacheDir) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(2000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, errx.User("opening index: %v", err)
	}
	idx := &Index{db: db, cacheDir: cacheDir}
	if err := idx.ensureSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return idx, nil
}

func (i *Index) Close() error { return i.db.Close() }

func discard(cacheDir string) {
	os.Remove(dbPath(cacheDir))
	os.Remove(dbPath(cacheDir) + "-wal")
	os.Remove(dbPath(cacheDir) + "-shm")
	os.Remove(metaPath(cacheDir))
}
