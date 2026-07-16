package core

import (
	"fmt"
	"maps"
	"slices"
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
	Source     datamodel.ChangeSource
}

func (s *Store) Move(cfg *datamodel.Config, ref, state string, opts MoveOpts) (*datamodel.MoveResult, error) {
	var from string
	var wipWarnings []string
	apply := func(it *datamodel.Item, _ *id.Resolver, items []*datamodel.Item) ([]error, []error) {
		from = it.State
		it.State = state
		hard, warns, wipWarns := applyStateChange(cfg, it, from, opts, nil, items)
		if len(hard) > 0 {
			return hard, nil
		}
		for _, e := range wipWarns {
			wipWarnings = append(wipWarnings, e.Error())
		}
		return nil, append(warns, wipWarns...)
	}
	subjectOf := func(orig *datamodel.Item) string {
		return fmt.Sprintf(cfg.Commit.SubjectPrefix+"%s state %s -> %s", orig.Number, orig.State, state)
	}

	source := opts.Source
	if source == "" {
		source = datamodel.SourceCLI
	}
	updated, _, err := s.mutate(cfg, ref, opts.Force, apply, subjectOf, source)
	if err != nil {
		return nil, err
	}
	if opts.Activate {
		if err := s.setActive(updated.ID); err != nil {
			return nil, err
		}
	}
	return &datamodel.MoveResult{
		ID:        updated.ID,
		Number:    updated.Number,
		From:      from,
		To:        updated.State,
		Activated: opts.Activate,
		Warnings:  wipWarnings,
	}, nil
}

func applyStateChange(cfg *datamodel.Config, it *datamodel.Item, from string, opts MoveOpts, edited map[string]bool, items []*datamodel.Item) (hard, warns, wipWarns []error) {
	state := it.State
	wf, ok := cfg.Workflows[it.Type]
	if !ok {
		return []error{fmt.Errorf("no workflow configured for type %q", it.Type)}, nil, nil
	}
	target, ok := wf.StateByKey(state)
	if !ok {
		return []error{errx.User("%q is not a state in the %s workflow", state, it.Type).WithHint("%s", stateHint(wf, state))}, nil, nil
	}
	tr := matchedTransition(wf, from, state)
	if wf.EnforceTransitions && from != state && tr == nil {
		if !opts.Force {
			return []error{errx.User("%s -> %s is not an allowed transition", from, state).WithHint("%s", transitionHint(wf, from))}, nil, nil
		}
		warns = append(warns, fmt.Errorf("forced off-graph transition %s -> %s", from, state))
	}
	if opts.Resolution != "" {
		if target.Category != datamodel.CategoryDone {
			return []error{fmt.Errorf("field %q: cannot be set because %s is not a done-category state", datamodel.KeyResolution, state)}, nil, nil
		}
		it.Resolution = &opts.Resolution
	}
	if tr != nil {
		h, w := applyTransitionEffects(cfg, it, tr, from, state, opts, edited, items)
		if len(h) > 0 {
			return h, nil, nil
		}
		warns = append(warns, w...)
	}

	if target.Category == datamodel.CategoryDone {
		explicit := opts.Resolution != "" || edited[datamodel.KeyResolution] || (tr != nil && tr.Set[datamodel.KeyResolution] != "")
		if !explicit && target.Resolution != "" {
			it.Resolution = &target.Resolution
		}
	} else {
		it.Resolution = nil
	}

	h, w := wipGuard(wf, it, items, from, state, target.Wip, opts.Force)
	if len(h) > 0 {
		return h, nil, nil
	}
	return nil, warns, w
}

func applyTransitionEffects(cfg *datamodel.Config, it *datamodel.Item, tr *datamodel.Transition, from, state string, opts MoveOpts, edited map[string]bool, items []*datamodel.Item) (hard, warns []error) {
	var missing []string
	for _, f := range tr.Require {
		if f == datamodel.RequireBlockersClosed {
			continue
		}
		if !fieldPresent(it, f) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		fields := strings.Join(missing, ", ")
		if !opts.Force {
			return []error{errx.User("%s -> %s requires %s to be set", from, state, fields).WithHint("set %s first, or use `--force` to override", fields)}, nil
		}
		warns = append(warns, fmt.Errorf("forced past require guard: %s not set", fields))
	}
	if slices.Contains(tr.Require, datamodel.RequireBlockersClosed) {
		h, w := blockersClosedGuard(cfg, it, items, from, state, opts.Force)
		if len(h) > 0 {
			return h, nil
		}
		warns = append(warns, w...)
	}
	for _, f := range slices.Sorted(maps.Keys(tr.Set)) {
		if edited[f] || (f == datamodel.KeyResolution && opts.Resolution != "") {
			continue
		}
		if err := applyFieldEdit(it, f, tr.Set[f]); err != nil {
			return []error{err}, nil
		}
	}
	return nil, warns
}

func wipGuard(wf datamodel.Workflow, it *datamodel.Item, items []*datamodel.Item, from, state string, wip int, force bool) (hard, warns []error) {
	if from == state || wip <= 0 {
		return nil, nil
	}
	inTargetState := 1
	for _, other := range items {
		if other.ID != it.ID && other.Type == it.Type && other.State == state {
			inTargetState++
		}
	}
	if inTargetState <= wip {
		return nil, nil
	}
	if wf.EffectiveWipPolicy() == datamodel.WipBlock && !force {
		return []error{errx.User("%s is over its WIP limit (%d > %d)", state, inTargetState, wip).WithHint("move an item out of %s first, or use `--force` to override", state)}, nil
	}
	return nil, []error{fmt.Errorf("%s is over its WIP limit (%d > %d)", state, inTargetState, wip)}
}

func blockersClosedGuard(cfg *datamodel.Config, it *datamodel.Item, items []*datamodel.Item, from, state string, force bool) (hard, warns []error) {
	byID := byULID(items)
	open, skipped := query.OpenBlockers(cfg, it, byID)
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
