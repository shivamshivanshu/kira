// Package storage reads and writes kira item files under .kira/, handling file locking and ticket-path resolution.
package storage

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/errx"
)

const (
	DirName          = ".kira"
	itemsDirName     = "tickets"
	templatesDirName = "templates"
	schemaDirName    = "schema"
	CacheDirName     = ".cache"
	configFileName   = "config.yaml"
	itemExt          = ".md"

	TicketsPrefix = DirName + "/" + itemsDirName
	ConfigRelPath = DirName + "/" + configFileName
)

var ErrStoreNotFound = errors.New("no .kira/ found")

type FS struct {
	root string
}

func New(root string) *FS { return &FS{root: root} }

func Discover(startDir string) (*FS, error) {
	dir := startDir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, errx.Env("cannot determine working directory: %v", err)
		}
		dir = cwd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, errx.Env("resolving %q: %v", dir, err)
	}
	for {
		if fi, err := os.Stat(filepath.Join(abs, DirName)); err == nil && fi.IsDir() {
			root, err := filepath.EvalSymlinks(abs)
			if err != nil {
				return nil, errx.Env("resolving %q: %v", abs, err)
			}
			return &FS{root: root}, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return nil, errx.Env("%w in %q or any parent directory", ErrStoreNotFound, dir).WithHint("run `kira init` to create one here")
		}
		abs = parent
	}
}

func (s *FS) Root() string        { return s.root }
func (s *FS) KiraDir() string     { return filepath.Join(s.root, DirName) }
func (s *FS) ConfigPath() string  { return filepath.Join(s.KiraDir(), configFileName) }
func (s *FS) ItemsDir() string    { return filepath.Join(s.KiraDir(), itemsDirName) }
func (s *FS) TemplateDir() string { return filepath.Join(s.KiraDir(), templatesDirName) }
func (s *FS) SchemaDir() string   { return filepath.Join(s.KiraDir(), schemaDirName) }
func (s *FS) CacheDir() string    { return filepath.Join(s.KiraDir(), CacheDirName) }

func ItemFilename(ulid string) string { return ulid + itemExt }

func (s *FS) ItemPath(ulid string) string {
	return filepath.Join(s.ItemsDir(), ItemFilename(ulid))
}

func (s *FS) RelToRoot(abs string) string {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return abs
	}
	return rel
}
