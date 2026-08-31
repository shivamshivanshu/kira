package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/query"
)

// Snapshot shares a current item index and query context across related reads.
// It is request-scoped; item details and Git history are read when requested.
type Snapshot struct {
	store *Store
	data  *loaded
	query query.Options
}

func (s *Store) Snapshot(cfg *datamodel.Config) (*Snapshot, error) {
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	qopts, err := s.queryOptions(ld.cfg, ld.resolver, ld.items)
	if err != nil {
		return nil, err
	}
	return &Snapshot{store: s, data: ld, query: qopts}, nil
}

func (v *Snapshot) List(opts ListOpts) (*datamodel.ListResult, error) {
	if opts.At != "" {
		return nil, errx.User("a current snapshot cannot query historical items")
	}
	return listResult(v.data, opts, v.query)
}

func (v *Snapshot) Board(opts BoardOpts) (*datamodel.BoardResult, error) {
	if opts.At != "" {
		return nil, errx.User("a current snapshot cannot query historical items")
	}
	return boardResult(v.data, opts, v.query)
}

func (v *Snapshot) Tree(ref string) (*datamodel.TreeResult, error) {
	return treeResult(v.data, ref)
}

func (v *Snapshot) SprintList() *datamodel.SprintListResult {
	return sprintList(v.data.cfg, v.data.items, v.query.ActiveSprint)
}

func (v *Snapshot) EpicProgress() (map[string]datamodel.EpicProgress, error) {
	return allEpicProgress(v.data.cfg, v.data.items)
}

func (v *Snapshot) Now() (*datamodel.NowResult, error) {
	if err := v.store.requireRepo(); err != nil {
		return nil, err
	}
	return v.store.now(v.data.cfg, v.data)
}

func (v *Snapshot) Detail(ref string) (*ItemDetail, error) {
	return v.store.detail(v.data, ref)
}
