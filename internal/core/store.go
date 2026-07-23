package core

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
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

// DiscoverGitInvoked is for hook shims and the merge driver: git always runs
// these with startDir at the repo toplevel, regardless of where the user
// actually ran the git command. GIT_PREFIX carries that real location as a
// toplevel-relative path, letting this find a .kira nested below toplevel.
func DiscoverGitInvoked(startDir string, prompter ...Prompter) (*Store, error) {
	if prefix := os.Getenv("GIT_PREFIX"); prefix != "" {
		startDir = filepath.Join(startDir, prefix)
	}
	return Discover(startDir, prompter...)
}

func (s *Store) WithPrompter(p Prompter) *Store {
	c := *s
	c.prompter = firstPrompter([]Prompter{p})
	return &c
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

type loadResult struct {
	items    []*datamodel.Item
	snap     id.Snapshot
	resolver *id.Resolver
	warnings []string
}

func (s *Store) load(cfg *datamodel.Config) (loadResult, error) {
	items, warnings, err := s.LoadAll()
	if err != nil {
		return loadResult{}, err
	}
	for _, it := range items {
		it.Activity = it.Updated
	}
	snap, resolver := storage.SnapshotAndResolver(cfg.Project.Key, items)
	return loadResult{items: items, snap: snap, resolver: resolver, warnings: warnings}, nil
}

func (s *Store) resolveRef(cfg *datamodel.Config, ref string) (*datamodel.Item, []*datamodel.Item, *id.Resolver, error) {
	ld, err := s.load(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	it, err := findItem(ld.items, ld.resolver, ref)
	if err != nil {
		return nil, nil, nil, err
	}
	return it, ld.items, ld.resolver, nil
}

func findItem(items []*datamodel.Item, resolver *id.Resolver, ref string) (*datamodel.Item, error) {
	ulid, err := resolveID(resolver, ref)
	if err != nil {
		return nil, err
	}
	if it := findByULID(items, ulid); it != nil {
		return it, nil
	}
	return nil, errx.User("resolved %s to %s, which has no file", ref, ulid)
}

func (s *Store) resolveMe(cfg *datamodel.Config, value string) (string, error) {
	if value != query.MeToken {
		return value, nil
	}
	if id, ok := s.identity(cfg); ok {
		return id, nil
	}
	return "", errx.User("cannot resolve @me: set git user.name or user.email")
}

func (s *Store) identity(cfg *datamodel.Config) (string, bool) {
	name := strings.TrimSpace(s.gitConfig("user.name"))
	email := strings.TrimSpace(s.gitConfig("user.email"))
	if canon, ok := cfg.People.Canonical(name, email); ok {
		return canon, true
	}
	for _, v := range []string{name, email} {
		if f := strings.Fields(v); len(f) > 0 {
			return strings.Join(f, "-"), true
		}
	}
	return "", false
}

func (s *Store) gitConfig(key string) string {
	v, err := s.repo().Output("config", key)
	if err != nil {
		return ""
	}
	return v
}

func (s *Store) RefExists(cfg *datamodel.Config, ref string) bool {
	_, _, _, err := s.resolveRef(cfg, ref)
	return err == nil
}

func (s *Store) ResolveItemFile(cfg *datamodel.Config, ref string) (string, string, error) {
	it, _, _, err := s.resolveRef(cfg, ref)
	if err != nil {
		return "", "", err
	}
	path := s.itemPath(it.ID)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", errx.User("reading %s: %v", ref, err)
	}
	return path, string(data), nil
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
	return datamodel.IndexByID(items)
}

func findByULID(items []*datamodel.Item, ulid string) *datamodel.Item {
	for _, it := range items {
		if it.ID == ulid {
			return it
		}
	}
	return nil
}

func replaceByULID(items []*datamodel.Item, updated *datamodel.Item) {
	for i, it := range items {
		if it.ID == updated.ID {
			items[i] = updated
			return
		}
	}
}
