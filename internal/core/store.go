package core

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// dirName is the tracked kira directory at the repo root.
const dirName = ".kira"

// Store is the handle to one repository's .kira/ tree. It owns filesystem
// discovery, atomic ticket writes, the advisory lock, and git invocation. All
// mutation goes through it, so the single-write-path guarantee
// (docs/design/01-architecture.md §6) has exactly one implementation.
type Store struct {
	// root is the absolute directory containing .kira/ (also the git work tree
	// root in the normal layout). git commands run here and staged paths are
	// expressed relative to it.
	root string
}

// Root returns the absolute directory that contains .kira/.
func (s *Store) Root() string { return s.root }

func (s *Store) kiraDir() string     { return filepath.Join(s.root, dirName) }
func (s *Store) configPath() string  { return filepath.Join(s.kiraDir(), "config.yaml") }
func (s *Store) ticketsDir() string  { return filepath.Join(s.kiraDir(), "tickets") }
func (s *Store) templateDir() string { return filepath.Join(s.kiraDir(), "templates") }
func (s *Store) cacheDir() string    { return filepath.Join(s.kiraDir(), ".cache") }

// itemPath returns the on-disk path of the item with the given ULID.
func (s *Store) itemPath(ulid string) string {
	return filepath.Join(s.ticketsDir(), ulid+".md")
}

// relToRoot expresses an absolute path under the store relative to root, the
// form git add/commit expect.
func (s *Store) relToRoot(abs string) string {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return abs
	}
	return rel
}

// Discover walks up from startDir to the first ancestor containing a .kira/
// directory, mirroring git's own repo discovery. startDir empty means the
// current working directory. It is the entry point for every command except
// init; a missing .kira/ is an environment error (exit 3).
func Discover(startDir string) (*Store, error) {
	dir := startDir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, envErr("cannot determine working directory: %v", err)
		}
		dir = cwd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, envErr("resolving %q: %v", dir, err)
	}
	for {
		if fi, err := os.Stat(filepath.Join(abs, dirName)); err == nil && fi.IsDir() {
			return &Store{root: abs}, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return nil, envErr("no %s/ found in %q or any parent directory (run `kira init`)", dirName, dir)
		}
		abs = parent
	}
}

// Config loads and validates this store's config.yaml.
func (s *Store) Config() (*config.Config, error) {
	cfg, err := config.Load(s.root)
	if err != nil {
		return nil, userErr("%v", err)
	}
	return cfg, nil
}

// LoadAll reads and parses every ticket file under tickets/, in ULID order. A
// single malformed file fails the whole load: a corrupt canonical file is a
// real error, not a row to skip silently.
func (s *Store) LoadAll() ([]*item.Item, error) {
	entries, err := os.ReadDir(s.ticketsDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // no tickets/ yet: an empty, freshly-initialized store
		}
		return nil, userErr("reading tickets: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	items := make([]*item.Item, 0, len(names))
	for _, name := range names {
		path := filepath.Join(s.ticketsDir(), name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, userErr("reading %s: %v", name, err)
		}
		it, err := item.Parse(string(data))
		if err != nil {
			return nil, userErr("parsing %s: %v", name, err)
		}
		items = append(items, it)
	}
	return items, nil
}

// load reads every item once and returns it alongside the identity snapshot and
// resolver built from it — the common preamble of create, show, edit, and list,
// so no command scans tickets/ twice.
func (s *Store) load(cfg *config.Config) ([]*item.Item, id.Snapshot, *id.Resolver, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, id.Snapshot{}, nil, err
	}
	snap := snapshot(cfg.Project.Key, items)
	return items, snap, id.NewResolver(snap), nil
}

// resolveRef loads the store, resolves ref (ULID | prefix | number | alias) to
// its item, and returns the resolver built from the same scan. The resolver is
// returned so a mutating caller (edit) can normalize the resolved item's
// cross-references without a second scan; read-only callers ignore it.
func (s *Store) resolveRef(cfg *config.Config, ref string) (*item.Item, *id.Resolver, error) {
	items, _, resolver, err := s.load(cfg)
	if err != nil {
		return nil, nil, err
	}
	ulid, err := resolver.Resolve(ref)
	if err != nil {
		return nil, nil, userErr("%v", err)
	}
	if it := findByULID(items, ulid); it != nil {
		return it, resolver, nil
	}
	return nil, nil, userErr("resolved %s to %s, which has no file", ref, ulid)
}

// findByULID returns the loaded item with the given canonical ULID, or nil.
func findByULID(items []*item.Item, ulid string) *item.Item {
	for _, it := range items {
		if it.ID == ulid {
			return it
		}
	}
	return nil
}

// snapshot builds the id.Snapshot (identity projection) the resolver and
// allocator consume, from an already-loaded item set and the project key.
func snapshot(key string, items []*item.Item) id.Snapshot {
	snap := id.Snapshot{Key: key, Items: make([]id.Item, len(items))}
	for i, it := range items {
		snap.Items[i] = id.Item{ULID: it.ID, Number: it.Number, Aliases: it.Aliases}
	}
	return snap
}

// writeItem serializes an item to tickets/<ulid>.md via the atomic raw writer.
func (s *Store) writeItem(it *item.Item) (string, error) {
	return s.writeItemRaw(it.ID, it.Serialize())
}

// writeItemRaw writes content to tickets/<ulid>.md via temp-file + fsync +
// rename, so a reader never observes a partially written file
// (docs/design/03-storage-and-git.md §4). It returns the repo-relative path for
// staging. The comment path writes pre-rendered content here (a pure byte-suffix
// append) rather than reserializing an item, which would rewrite frontmatter.
func (s *Store) writeItemRaw(ulid, content string) (string, error) {
	if err := os.MkdirAll(s.ticketsDir(), 0o755); err != nil {
		return "", userErr("creating tickets dir: %v", err)
	}
	dst := s.itemPath(ulid)
	tmp := filepath.Join(s.ticketsDir(), "."+ulid+".md.tmp")
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return "", userErr("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", userErr("writing temp file: %v", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", userErr("syncing temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", userErr("closing temp file: %v", err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return "", userErr("renaming into place: %v", err)
	}
	return s.relToRoot(dst), nil
}

// lock takes the advisory lock at .cache/lock, serializing concurrent kira
// mutations against this repo (docs/design/03-storage-and-git.md §4, §5). It
// blocks up to ~2s, then fails loudly rather than hang. The returned release
// must be called to unlock. The lock file is cache-adjacent (gitignored) and is
// purely local-process coordination, not a git lock.
func (s *Store) lock() (func(), error) {
	if err := os.MkdirAll(s.cacheDir(), 0o755); err != nil {
		return nil, userErr("creating cache dir: %v", err)
	}
	path := filepath.Join(s.cacheDir(), "lock")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, userErr("opening lock: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			return func() {
				syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
				f.Close()
			}, nil
		}
		if time.Now().After(deadline) {
			f.Close()
			return nil, conflictErr("another kira process holds the lock on %s", s.root)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
