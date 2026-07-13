package core

import (
	"os"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type Store struct {
	root     string
	store    *storage.FS
	prompter Prompter
}

func newStore(root string) *Store {
	return &Store{root: root, store: storage.New(root), prompter: silentPrompter{}}
}

func Discover(startDir string, prompter ...Prompter) (*Store, error) {
	store, err := storage.Discover(startDir)
	if err != nil {
		return nil, err
	}
	return &Store{root: store.Root(), store: store, prompter: firstPrompter(prompter)}, nil
}

func (s *Store) fs() *storage.FS { return s.store }

func (s *Store) Root() string { return s.root }

func (s *Store) Config() (*datamodel.Config, error) {
	cfg, err := config.LoadWithUser(s.root, os.Getenv, os.Stderr)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	return cfg, nil
}

func (s *Store) LoadAll() ([]*datamodel.Item, []string, error) { return s.fs().LoadAll() }

func (s *Store) itemPath(ulid string) string { return s.fs().ItemPath(ulid) }

func (s *Store) load(cfg *datamodel.Config) ([]*datamodel.Item, id.Snapshot, *id.Resolver, []string, error) {
	items, warnings, err := s.LoadAll()
	if err != nil {
		return nil, id.Snapshot{}, nil, nil, err
	}
	snap, resolver := snapshotAndResolver(cfg.Project.Key, items)
	return items, snap, resolver, warnings, nil
}

func (s *Store) resolveRef(cfg *datamodel.Config, ref string) (*datamodel.Item, []*datamodel.Item, *id.Resolver, error) {
	items, _, resolver, _, err := s.load(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	ulid, err := resolveID(resolver, ref)
	if err != nil {
		return nil, nil, nil, err
	}
	if it := findByULID(items, ulid); it != nil {
		return it, items, resolver, nil
	}
	return nil, nil, nil, errx.User("resolved %s to %s, which has no file", ref, ulid)
}

func guardWritable(items ...*datamodel.Item) error {
	for _, it := range items {
		if it == nil {
			continue
		}
		if it.HasUnknown() {
			names := slices.Concat(it.UnknownKeys, it.UnknownLinkTypes)
			return errx.Env("this item uses fields from a newer kira: %s", strings.Join(names, ", ")).
				WithHint("upgrade kira, then retry: `go install github.com/shivamshivanshu/kira/cmd/kira@latest`")
		}
		if it.CRLF {
			return errx.Env("this item has CRLF line endings, which kira cannot rewrite byte-stably").
				WithHint("renormalize to LF, e.g. `git add --renormalize .` with `.kira/** text eol=lf` in .gitattributes")
		}
	}
	return nil
}

func byULID(items []*datamodel.Item) map[string]*datamodel.Item {
	m := make(map[string]*datamodel.Item, len(items))
	for _, it := range items {
		m[it.ID] = it
	}
	return m
}

func findByULID(items []*datamodel.Item, ulid string) *datamodel.Item {
	for _, it := range items {
		if it.ID == ulid {
			return it
		}
	}
	return nil
}
