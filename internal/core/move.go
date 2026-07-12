package core

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// MoveOpts are the move flags (docs/design/04-cli.md move). Resolution, when
// non-empty, is recorded on the item's resolution field (vocab-validated) and
// outranks a transition set: and the target state's resolution: tag; Force
// bypasses the adjacency check and any require: guard, never state existence.
type MoveOpts struct {
	Resolution string
	Force      bool
	Activate   bool
}

// MoveResult reports the transition. The shape is pinned by docs/design/04-cli.md
// §7; Warnings is the additive key carrying WIP-limit breaches, absent otherwise.
type MoveResult struct {
	ID        string   `json:"id"`
	Number    string   `json:"number"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Activated bool     `json:"activated"`
	Warnings  []string `json:"warnings,omitempty"`
}

// Move transitions ref to state, enforcing the item type's workflow
// (docs/design/04-cli.md move; docs/design/02-data-model.md §6): the adjacency
// check, the matched transition's require: guard (both --force-bypassable with
// a warning), its set: assignments, the resolution done-entry/exit lifecycle,
// and the advisory WIP warning. An unknown state name is always an error, even
// under --force. With --activate it additionally records ref as the active
// ticket.
func (s *Store) Move(cfg *config.Config, ref, state string, opts MoveOpts) (*MoveResult, error) {
	var from string
	var wipWarnings []string
	apply := func(it *item.Item, _ *id.Resolver, items []*item.Item) (hard, warns []error) {
		wf, ok := cfg.Workflows[it.Type]
		if !ok {
			return []error{fmt.Errorf("no workflow configured for type %q", it.Type)}, nil
		}
		target, ok := stateIn(wf, state)
		if !ok {
			return []error{fmt.Errorf("%q is not a state in the %s workflow", state, it.Type)}, nil
		}
		from = it.State
		it.State = state
		tr := matchedTransition(wf, from, state)
		if wf.EnforceTransitions && from != state && tr == nil {
			if !opts.Force {
				return []error{fmt.Errorf("%s -> %s is not an allowed transition (use --force to override)", from, state)}, nil
			}
			warns = append(warns, fmt.Errorf("forced off-graph transition %s -> %s", from, state))
		}
		if opts.Resolution != "" {
			if target.Category != config.CategoryDone {
				return []error{fmt.Errorf("--resolution: %s is not a done-category state", state)}, nil
			}
			it.Resolution = &opts.Resolution
		}
		if tr != nil {
			// require: is checked before set: is applied, so a transition
			// cannot satisfy its own guard (docs/design/04-cli.md move).
			var missing []string
			for _, f := range tr.Require {
				if !fieldPresent(it, f) {
					missing = append(missing, f)
				}
			}
			if len(missing) > 0 {
				fields := strings.Join(missing, ", ")
				if !opts.Force {
					return []error{fmt.Errorf("%s -> %s requires %s to be set (use --force to override)", from, state, fields)}, nil
				}
				warns = append(warns, fmt.Errorf("forced past require guard: %s not set", fields))
			}
			for _, f := range slices.Sorted(maps.Keys(tr.Set)) {
				if f == "resolution" && opts.Resolution != "" {
					continue // --resolution outranks the transition's set:
				}
				if err := applyFieldEdit(it, f, tr.Set[f]); err != nil {
					return []error{err}, nil
				}
			}
		}

		// Resolution lifecycle (docs/design/02-data-model.md §6): entering a
		// done-category state takes the state's resolution: tag when nothing in
		// this move set the field; leaving done-category clears it.
		if target.Category == config.CategoryDone {
			explicit := opts.Resolution != "" || (tr != nil && tr.Set["resolution"] != "")
			if !explicit && target.Resolution != "" {
				it.Resolution = &target.Resolution
			}
		} else {
			it.Resolution = nil
		}

		// WIP is advisory: warn on breach, never block (02 §6). The census is
		// per workflow type, from the same scan that resolved ref.
		if from != state && target.Wip > 0 {
			count := 1 // the item being moved
			for _, other := range items {
				if other.ID != it.ID && other.Type == it.Type && other.State == state {
					count++
				}
			}
			if count > target.Wip {
				w := fmt.Errorf("%s is over its WIP limit (%d > %d)", state, count, target.Wip)
				wipWarnings = append(wipWarnings, w.Error())
				warns = append(warns, w)
			}
		}
		return nil, warns
	}
	subjectOf := func(orig *item.Item) string {
		return fmt.Sprintf("kira: %s state %s -> %s", orig.Number, orig.State, state)
	}

	updated, _, err := s.mutate(cfg, ref, opts.Force, apply, subjectOf)
	if err != nil {
		return nil, err
	}
	if opts.Activate {
		if err := s.setActive(updated.ID); err != nil {
			return nil, err
		}
	}
	return &MoveResult{
		ID:        updated.ID,
		Number:    updated.Number,
		From:      from,
		To:        updated.State,
		Activated: opts.Activate,
		Warnings:  wipWarnings,
	}, nil
}

// setActive records ulid as the active ticket at .cache/active. The pointer
// drives the prepare-commit-msg trailer auto-insert (docs/design/04-cli.md
// move); it lives under .cache/ so it is gitignored, never tracked state. The
// directory already exists: every Move holds the lock, which creates .cache/.
func (s *Store) setActive(ulid string) error {
	if err := os.WriteFile(filepath.Join(s.cacheDir(), "active"), []byte(ulid+"\n"), 0o644); err != nil {
		return userErr("writing active pointer: %v", err)
	}
	return nil
}
