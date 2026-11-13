package core

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
	"github.com/shivamshivanshu/kira/internal/workon"
)

type MoveOpts struct {
	Resolution string
	Force      bool
	Activate   bool
}

func (s *Store) Move(cfg *datamodel.Config, ref, state string, opts MoveOpts) (*datamodel.MoveResult, error) {
	b, err := s.BeginBatch(cfg)
	if err != nil {
		return nil, err
	}
	defer b.Close()
	return b.Move(ref, state, opts)
}

func (b *Batch) Move(ref, state string, opts MoveOpts) (*datamodel.MoveResult, error) {
	cfg := b.cfg
	var from string
	apply := func(it *datamodel.Item, _ *id.Resolver, items []*datamodel.Item) ([]error, []error) {
		from = it.State
		it.State = state
		hard, warns, wipWarns := applyStateChange(it, from, opts, nil, items)
		if len(hard) > 0 {
			return hard, nil
		}
		return nil, append(warns, wipWarns...)
	}
	subjectOf := func(orig *datamodel.Item) string {
		return fmt.Sprintf("%s%s state %s -> %s", cfg.Commit.SubjectPrefix, orig.Number, orig.State, state)
	}

	updated, _, warns, err := b.Mutate(ref, opts.Force, apply, subjectOf)
	if err != nil {
		return nil, err
	}
	if opts.Activate {
		if err := b.store.setActive(updated.ID); err != nil {
			return nil, err
		}
	}
	return &datamodel.MoveResult{
		ID:        updated.ID,
		Number:    updated.Number,
		From:      from,
		To:        updated.State,
		Activated: opts.Activate,
		Warnings:  warningStrings(warns),
	}, nil
}

func applyStateChange(it *datamodel.Item, from string, opts MoveOpts, edited map[string]bool, items []*datamodel.Item) (hard, warns, wipWarns []error) {
	state := it.State
	wf, _ := datamodel.WorkflowOf(it.Type)
	target, ok := wf.StateByKey(state)
	if !ok {
		return []error{errx.User("%q is not a state in the %s workflow", state, it.Type).WithHint("%s", stateHint(wf, state))}, nil, nil
	}
	if opts.Resolution != "" {
		if target.Category != datamodel.CategoryDone {
			return []error{fmt.Errorf("field %q: cannot be set because %s is not a done-category state", datamodel.KeyResolution, state)}, nil, nil
		}
		it.Resolution = &opts.Resolution
	}

	if target.Category == datamodel.CategoryDone {
		if from != state {
			h, w := blockersClosedGuard(it, items, from, state, opts.Force)
			if len(h) > 0 {
				return h, nil, nil
			}
			warns = append(warns, w...)
		}
		if opts.Resolution == "" && !edited[datamodel.KeyResolution] && target.Resolution != "" {
			it.Resolution = &target.Resolution
		}
	} else {
		it.Resolution = nil
	}

	return nil, warns, wipOverrunWarnings(it, items, from, state, target.Wip)
}

func wipOverrunWarnings(it *datamodel.Item, items []*datamodel.Item, from, state string, wip int) []error {
	if from == state || wip <= 0 {
		return nil
	}
	inTargetState := 1
	for _, other := range items {
		if other.ID != it.ID && other.Type == it.Type && other.State == state {
			inTargetState++
		}
	}
	if inTargetState <= wip {
		return nil
	}
	return []error{fmt.Errorf("%s is over its WIP limit (%d > %d)", state, inTargetState, wip)}
}

func blockersClosedGuard(it *datamodel.Item, items []*datamodel.Item, from, state string, force bool) (hard, warns []error) {
	byID := datamodel.IndexByID(items)
	open, skipped := query.OpenBlockers(it, byID)
	for _, s := range skipped {
		warns = append(warns, fmt.Errorf("blocked_by %s %s; treating blocker as satisfied", numberOrID(s.Blocker, s.ID), s.Reason))
	}
	if len(open) > 0 {
		refs := make([]string, len(open))
		for i, b := range open {
			refs[i] = numberOrID(b, b.ID)
		}
		joined := strings.Join(refs, ", ")
		if !force {
			return []error{errx.User("%s -> %s is blocked by open items: %s", from, state, joined).WithHint("close the blockers first, or use `--force` to override")}, nil
		}
		warns = append(warns, fmt.Errorf("forced past blocker guard: %s still open", joined))
	}
	return nil, warns
}

func (s *Store) setActive(ulid string) error {
	branch, _ := s.repo().CurrentBranch()
	return s.writeActive(workon.ActivePointer{Ticket: ulid, Branch: branch})
}
