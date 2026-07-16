package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

type AssignOpts struct {
	Reporter bool
	Force    bool
}

func (s *Store) Assign(cfg *datamodel.Config, ref, user string, opts AssignOpts) (*datamodel.MutationResult, error) {
	b, err := s.BeginBatch(cfg)
	if err != nil {
		return nil, err
	}
	defer b.Close()
	return b.Assign(ref, user, opts)
}

func (b *Batch) Assign(ref, user string, opts AssignOpts) (*datamodel.MutationResult, error) {
	cfg := b.cfg
	user, err := b.store.resolveMe(cfg, user)
	if err != nil {
		return nil, err
	}
	field := "owner"
	if opts.Reporter {
		field = "reporter"
	}
	apply := func(it *datamodel.Item, _ *id.Resolver, _ []*datamodel.Item) (hard, warns []error) {
		target := &it.Owner
		if opts.Reporter {
			target = &it.Reporter
		}
		*target = ptr.NilIfEmpty(user)
		return nil, nil
	}
	subjectOf := func(orig *datamodel.Item) string {
		return cfg.Commit.SubjectPrefix + orig.Number + " assign " + field + " " + user
	}

	updated, changed, err := b.Mutate(ref, opts.Force, apply, subjectOf, datamodel.SourceCLI)
	if err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}
