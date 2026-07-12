package core

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// MoveOpts are the move flags (docs/design/04-cli.md move).
type MoveOpts struct {
	Force    bool
	Activate bool
}

// MoveResult reports the transition. Its --json shape is not pinned by the doc
// set; this is the chosen contract.
type MoveResult struct {
	ID        string `json:"id"`
	Number    string `json:"number"`
	From      string `json:"from"`
	To        string `json:"to"`
	Activated bool   `json:"activated"`
}

// Move transitions ref to state, validating the move against the item type's
// workflow (docs/design/04-cli.md move; docs/design/02-data-model.md §6). An
// unknown state name is always an error, even under --force; --force only
// bypasses the adjacency check (with a warning) when enforce_transitions is on.
// With --activate it additionally records ref as the active ticket.
func (s *Store) Move(cfg *config.Config, ref, state string, opts MoveOpts) (*MoveResult, error) {
	var from string
	apply := func(it *item.Item, _ *id.Resolver) (hard, warns []error) {
		wf, ok := cfg.Workflows[it.Type]
		if !ok {
			return []error{fmt.Errorf("no workflow configured for type %q", it.Type)}, nil
		}
		if !stateInWorkflow(wf, state) {
			return []error{fmt.Errorf("%q is not a state in the %s workflow", state, it.Type)}, nil
		}
		from = it.State
		it.State = state
		if wf.EnforceTransitions && from != state && !slices.Contains(wf.Transitions[from], state) {
			if !opts.Force {
				return []error{fmt.Errorf("%s -> %s is not an allowed transition (use --force to override)", from, state)}, nil
			}
			warns = append(warns, fmt.Errorf("forced off-graph transition %s -> %s", from, state))
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
