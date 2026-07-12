package core

import (
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// AssignOpts are the assign flags (docs/design/04-cli.md assign).
type AssignOpts struct {
	Reporter bool // target the reporter field instead of owner
	Force    bool
}

// Assign sets ref's owner (or reporter, with --reporter) to user. The value is
// checked against people.known; an unknown value is rejected when people.strict
// and not --force, else warned (docs/design/02-data-model.md §5).
func (s *Store) Assign(cfg *config.Config, ref, user string, opts AssignOpts) (*MutationResult, error) {
	field := "owner"
	if opts.Reporter {
		field = "reporter"
	}
	apply := func(it *item.Item, _ *id.Resolver, _ []*item.Item) (hard, warns []error) {
		target := &it.Owner
		if opts.Reporter {
			target = &it.Reporter
		}
		*target = ptrOrNil(user)
		return nil, nil
	}
	subjectOf := func(orig *item.Item) string {
		return "kira: " + orig.Number + " assign " + field + " " + user
	}

	updated, changed, err := s.mutate(cfg, ref, opts.Force, apply, subjectOf)
	if err != nil {
		return nil, err
	}
	return &MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}
