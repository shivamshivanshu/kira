package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func (s *Store) Show(cfg *datamodel.Config, ref string) (*datamodel.ShowResult, error) {
	_, _, resolver, idxNotes, err := s.indexedLoad(cfg)
	if err != nil {
		return nil, err
	}
	ulid, err := resolver.Resolve(ref)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	it, err := storage.ReadItem(s.itemPath(ulid))
	if err != nil {
		return nil, errx.User("reading %s: %v", ref, err)
	}
	res := showResultOf(cfg, it)
	res.StderrNotes = idxNotes
	return &res, nil
}
