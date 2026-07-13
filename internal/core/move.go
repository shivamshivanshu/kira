package core

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
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
	apply := func(it *datamodel.Item, _ *id.Resolver, items []*datamodel.Item) (hard, warns []error) {
		wf, ok := cfg.Workflows[it.Type]
		if !ok {
			return []error{fmt.Errorf("no workflow configured for type %q", it.Type)}, nil
		}
		target, ok := stateIn(wf, state)
		if !ok {
			return []error{errx.User("%q is not a state in the %s workflow", state, it.Type).WithHint("%s", stateHint(wf, state))}, nil
		}
		from = it.State
		it.State = state
		tr := matchedTransition(wf, from, state)
		if wf.EnforceTransitions && from != state && tr == nil {
			if !opts.Force {
				return []error{errx.User("%s -> %s is not an allowed transition", from, state).WithHint("%s", transitionHint(wf, from))}, nil
			}
			warns = append(warns, fmt.Errorf("forced off-graph transition %s -> %s", from, state))
		}
		if opts.Resolution != "" {
			if target.Category != datamodel.CategoryDone {
				return []error{fmt.Errorf("--resolution: %s is not a done-category state", state)}, nil
			}
			it.Resolution = &opts.Resolution
		}
		if tr != nil {
			h, w := applyTransitionEffects(it, tr, from, state, opts)
			if len(h) > 0 {
				return h, nil
			}
			warns = append(warns, w...)
		}

		if target.Category == datamodel.CategoryDone {
			explicit := opts.Resolution != "" || (tr != nil && tr.Set["resolution"] != "")
			if !explicit && target.Resolution != "" {
				it.Resolution = &target.Resolution
			}
		} else {
			it.Resolution = nil
		}

		if from != state && target.Wip > 0 {
			inTargetState := 1
			for _, other := range items {
				if other.ID != it.ID && other.Type == it.Type && other.State == state {
					inTargetState++
				}
			}
			if inTargetState > target.Wip {
				w := fmt.Errorf("%s is over its WIP limit (%d > %d)", state, inTargetState, target.Wip)
				wipWarnings = append(wipWarnings, w.Error())
				warns = append(warns, w)
			}
		}
		return nil, warns
	}
	subjectOf := func(orig *datamodel.Item) string {
		return fmt.Sprintf(subjectPrefix+"%s state %s -> %s", orig.Number, orig.State, state)
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

func applyTransitionEffects(it *datamodel.Item, tr *datamodel.Transition, from, state string, opts MoveOpts) (hard, warns []error) {
	var missing []string
	for _, f := range tr.Require {
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
	for _, f := range slices.Sorted(maps.Keys(tr.Set)) {
		if f == "resolution" && opts.Resolution != "" {
			continue
		}
		if err := applyFieldEdit(it, f, tr.Set[f]); err != nil {
			return []error{err}, nil
		}
	}
	return nil, warns
}

func (s *Store) setActive(ulid string) error {
	branch, _ := s.repo().CurrentBranch()
	return s.writeActive(workon.ActivePointer{Ticket: ulid, Branch: branch})
}
