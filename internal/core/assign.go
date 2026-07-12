package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

type AssignOpts struct {
	Reporter bool
	Force    bool
}

func (s *Store) Assign(cfg *datamodel.Config, ref, user string, opts AssignOpts) (*datamodel.MutationResult, error) {
	field := "owner"
	if opts.Reporter {
		field = "reporter"
	}
	apply := func(it *datamodel.Item, _ *id.Resolver, _ []*datamodel.Item) (hard, warns []error) {
		target := &it.Owner
		if opts.Reporter {
			target = &it.Reporter
		}
		*target = ptrOrNil(user)
		return nil, nil
	}
	subjectOf := func(orig *datamodel.Item) string {
		return "kira: " + orig.Number + " assign " + field + " " + user
	}

	updated, changed, err := s.mutate(cfg, ref, opts.Force, apply, subjectOf)
	if err != nil {
		return nil, err
	}
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}
