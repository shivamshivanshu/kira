package core

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type Store struct {
	root  string
	store *storage.Store
}

func newStore(root string) *Store {
	return &Store{root: root, store: storage.New(root)}
}

func Discover(startDir string) (*Store, error) {
	fs, err := storage.Discover(startDir)
	if err != nil {
		return nil, err
	}
	return &Store{root: fs.Root(), store: fs}, nil
}

func (s *Store) fs() *storage.Store { return s.store }

func (s *Store) Root() string { return s.root }

func (s *Store) Config() (*datamodel.Config, error) {
	cfg, err := config.Load(s.root)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	return cfg, nil
}

func (s *Store) LoadAll() ([]*datamodel.Item, error) { return s.fs().LoadAll() }

func (s *Store) itemPath(ulid string) string { return s.fs().ItemPath(ulid) }

func (s *Store) writeItem(it *datamodel.Item) (string, error) { return s.fs().WriteItem(it) }

func (s *Store) writeItemRaw(ulid, content string) (string, error) {
	return s.fs().WriteItemRaw(ulid, content)
}

func (s *Store) load(cfg *datamodel.Config) ([]*datamodel.Item, id.Snapshot, *id.Resolver, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, id.Snapshot{}, nil, err
	}
	snap, resolver := resolverFor(cfg.Project.Key, items)
	return items, snap, resolver, nil
}

func (s *Store) resolveRef(cfg *datamodel.Config, ref string) (*datamodel.Item, []*datamodel.Item, *id.Resolver, error) {
	items, _, resolver, err := s.load(cfg)
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

func guardKnownFields(items ...*datamodel.Item) error {
	for _, it := range items {
		if it != nil && it.HasUnknown() {
			names := slices.Concat(it.UnknownKeys, it.UnknownLinkTypes)
			return errx.Env("this ticket uses fields from a newer kira: %s", strings.Join(names, ", ")).
				WithHint("upgrade kira, then retry: `go install github.com/shivamshivanshu/kira/cmd/kira@latest`")
		}
	}
	return nil
}

func findByULID(items []*datamodel.Item, ulid string) *datamodel.Item {
	for _, it := range items {
		if it.ID == ulid {
			return it
		}
	}
	return nil
}
