package core

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func (s *Store) listView(cfg *datamodel.Config, at string) ([]*datamodel.Item, *id.Resolver, *datamodel.Config, error) {
	if at == "" {
		items, _, resolver, err := s.load(cfg)
		return items, resolver, cfg, err
	}
	sha, err := s.resolveAtRef(at)
	if err != nil {
		return nil, nil, nil, err
	}
	loaded, err := treeish.Load(s.repo(), sha)
	if err != nil {
		return nil, nil, nil, errx.User("%v", err)
	}
	return loaded.Items, loaded.Resolver, loaded.Config, nil
}

func (s *Store) resolveAtRef(at string) (string, error) {
	repo := s.repo()
	if isDate(at) {
		sha, err := repo.ResolveAtDate(at, "HEAD")
		if err != nil {
			return "", errx.User("resolving --at %s: %v", at, err)
		}
		return sha, nil
	}
	sha, err := repo.ResolveTreeish(at)
	if err != nil {
		return "", errx.User("resolving --at %s: %v", at, err)
	}
	return sha, nil
}

func isDate(s string) bool {
	return datamodel.ValidDate(s)
}

func (s *Store) ShowView(cfg *datamodel.Config, ref, at string) (*datamodel.ShowResult, string, error) {
	if at == "" {
		res, err := s.Show(cfg, ref)
		return res, "", err
	}
	return s.ShowAt(cfg, ref, at)
}

func (s *Store) ShowAt(cfg *datamodel.Config, ref, at string) (*datamodel.ShowResult, string, error) {
	sha, err := s.resolveAtRef(at)
	if err != nil {
		return nil, "", err
	}
	loaded, err := treeish.Load(s.repo(), sha)
	if err != nil {
		return nil, "", errx.User("%v", err)
	}
	ulid, err := loaded.Resolver.Resolve(ref)
	if err != nil {
		return nil, "", errx.User("%v", err)
	}
	it := findByULID(loaded.Items, ulid)
	if it == nil {
		return nil, "", errx.User("%s resolved to %s, which is absent at %s", ref, ulid, at)
	}
	res := showResultOf(loaded.Config, it)
	return &res, s.skewNote(cfg, ref, ulid, at), nil
}

func (s *Store) skewNote(cfg *datamodel.Config, ref, atULID, at string) string {
	_, _, resolver, err := s.load(cfg)
	if err != nil {
		return ""
	}
	nowULID, err := resolver.Resolve(ref)
	if err != nil || nowULID == atULID {
		return ""
	}
	return fmt.Sprintf("%s at %s is %s; currently it is a different item (%s)", ref, at, atULID, nowULID)
}
