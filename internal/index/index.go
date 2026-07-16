// Package index maintains the disposable SQLite cache under .kira/.cache/; it is
// never authoritative and is fully rebuildable from the ticket files plus git.
package index

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // registers the "sqlite" driver used by sql.Open below

	"github.com/shivamshivanshu/kira/internal/errx"
)

// Index wraps the SQLite-backed ticket cache under a repo's cache dir.
type Index struct {
	db       *sql.DB
	cacheDir string
}

func dbPath(cacheDir string) string   { return filepath.Join(cacheDir, "index.db") }
func metaPath(cacheDir string) string { return filepath.Join(cacheDir, "meta.json") }

// Open opens the index cache under cacheDir, discarding and rebuilding it
// once from scratch if the existing cache fails to open (e.g. a schema
// mismatch or a corrupted database file).
func Open(cacheDir string) (*Index, error) {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, errx.Env("creating cache dir: %v", err)
	}
	idx, err := open(cacheDir)
	if err == nil {
		return idx, nil
	}
	if err := discard(cacheDir); err != nil {
		return nil, err
	}
	return open(cacheDir)
}

func open(cacheDir string) (*Index, error) {
	dsn := "file:" + dbPath(cacheDir) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(2000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, errx.Env("opening index: %v", err)
	}
	idx := &Index{db: db, cacheDir: cacheDir}
	if err := idx.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return idx, nil
}

// Close closes the index's underlying database connection.
func (i *Index) Close() error { return i.db.Close() }

// discard deletes the on-disk cache files so the next open rebuilds them from
// scratch. A missing file is not an error, but any other failure is: leaving
// a stale db/wal/shm behind here would make the retried open in Open reuse
// (rather than rebuild) whatever corrupted or mismatched state caused the
// discard in the first place, so it must abort instead of retrying blind.
func discard(cacheDir string) error {
	for _, path := range []string{
		dbPath(cacheDir),
		dbPath(cacheDir) + "-wal",
		dbPath(cacheDir) + "-shm",
		metaPath(cacheDir),
	} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return errx.Env("removing stale cache file %s: %v", path, err)
		}
	}
	return nil
}
