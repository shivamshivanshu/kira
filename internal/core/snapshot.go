package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
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
	ld.matches = make(map[string][]*datamodel.Item)
	return &Snapshot{store: s, data: ld, query: s.queryOptions(ld)}, nil
}

func (v *Snapshot) ListWithMatches(expr string) ([]datamodel.ListItem, map[string]bool, error) {
	return listWithMatches(v.data.cfg, v.data.items, expr, v.query)
}

func (v *Snapshot) Stats(opts StatsOpts) (*datamodel.StatsResult, error) {
	return v.store.stats(v.data, opts)
}

type ActiveItemSummary struct {
	ID, Number, Title, State, Category, Branch string
	BlockerCount                               int
}

// ActiveSummary supplies the dashboard banner without reading event history.
func (v *Snapshot) ActiveSummary() *ActiveItemSummary {
	branch, _ := v.store.repo().CurrentBranch()
	it, activeBranch, _ := v.store.activeItem(v.data.cfg, v.data, branch)
	if it == nil {
		return nil
	}
	return &ActiveItemSummary{ID: it.ID, Number: it.Number, Title: it.Title,
		State: it.State, Category: categoryString(v.data.cfg, it.Type, it.State),
		Branch: activeBranch, BlockerCount: len(it.BlockedBy)}
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
	return allEpicProgress(v.data.cfg, v.data.items, entityschema.ContainerTypes(v.data.schemas))
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
