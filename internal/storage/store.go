package storage

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/errx"
)

const dirName = ".kira"

var ErrStoreNotFound = errors.New("no .kira/ found")

type Store struct {
	root string
}

func New(root string) *Store { return &Store{root: root} }

func Discover(startDir string) (*Store, error) {
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
		if fi, err := os.Stat(filepath.Join(abs, dirName)); err == nil && fi.IsDir() {
			return &Store{root: abs}, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return nil, errx.Env("%w in %q or any parent directory", ErrStoreNotFound, dir).WithHint("run `kira init` to create one here")
		}
		abs = parent
	}
}

func (s *Store) Root() string        { return s.root }
func (s *Store) KiraDir() string     { return filepath.Join(s.root, dirName) }
func (s *Store) ConfigPath() string  { return filepath.Join(s.KiraDir(), "config.yaml") }
func (s *Store) ItemsDir() string    { return filepath.Join(s.KiraDir(), "tickets") }
func (s *Store) TemplateDir() string { return filepath.Join(s.KiraDir(), "templates") }
func (s *Store) CacheDir() string    { return filepath.Join(s.KiraDir(), ".cache") }

func (s *Store) ItemPath(ulid string) string {
	return filepath.Join(s.ItemsDir(), ulid+".md")
}

func (s *Store) RelToRoot(abs string) string {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return abs
	}
	return rel
}
