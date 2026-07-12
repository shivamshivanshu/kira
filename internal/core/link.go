package core

import (
	"fmt"
	"slices"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// LinkTarget selects which edge kira link operates on.
type LinkTarget int

// The two linkable edges. Both are single-sided: only the source item's field
// is written; blocks and epic-children are index-derived inverses, never stored
// (docs/design/02-data-model.md §3).
const (
	LinkEpic LinkTarget = iota
	LinkBlockedBy
)

// LinkOpts are the resolved link inputs (docs/design/04-cli.md link). The CLI
// picks Target from whichever of --epic/--blocked-by was given and rejects
// giving both or neither.
type LinkOpts struct {
	Target LinkTarget
	Ref    string // the other item's reference; may be empty only when removing an epic
	Remove bool
	Force  bool
}

// Link sets or removes one of ref's stored edges (docs/design/04-cli.md link).
// It writes only ref's own file — the inverse side (blocks, epic children) is
// derived, never stored. Self-links are rejected, and an epic may not be given
// an epic parent (epics are top-level).
func (s *Store) Link(cfg *config.Config, ref string, opts LinkOpts) (*MutationResult, error) {
	apply := func(it *item.Item, resolver *id.Resolver) (hard, warns []error) {
		switch opts.Target {
		case LinkEpic:
			return linkEpic(it, resolver, opts), nil
		default:
			return linkBlockedBy(it, resolver, opts), nil
		}
	}
	subjectOf := func(orig *item.Item) string {
		verb, edge := "link", edgeLabel(opts.Target)
		if opts.Remove {
			verb = "unlink"
		}
		if opts.Target == LinkEpic && opts.Remove {
			return fmt.Sprintf("kira: %s %s %s", orig.Number, verb, edge)
		}
		return fmt.Sprintf("kira: %s %s %s %s", orig.Number, verb, edge, opts.Ref)
	}

	updated, changed, err := s.mutate(cfg, ref, opts.Force, apply, subjectOf)
	if err != nil {
		return nil, err
	}
	return &MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}

func linkEpic(it *item.Item, resolver *id.Resolver, opts LinkOpts) []error {
	if opts.Remove {
		it.Epic = nil
		return nil
	}
	if it.Type == item.TypeEpic {
		return []error{fmt.Errorf("an epic cannot have an epic parent")}
	}
	ulid, err := resolveTarget(resolver, it, "epic", opts.Ref)
	if err != nil {
		return []error{err}
	}
	it.Epic = &ulid
	return nil
}

func linkBlockedBy(it *item.Item, resolver *id.Resolver, opts LinkOpts) []error {
	ulid, err := resolveTarget(resolver, it, "blocked_by", opts.Ref)
	if err != nil {
		return []error{err}
	}
	if opts.Remove {
		it.BlockedBy = slices.DeleteFunc(it.BlockedBy, func(b string) bool { return b == ulid })
		return nil
	}
	if !slices.Contains(it.BlockedBy, ulid) {
		it.BlockedBy = append(it.BlockedBy, ulid)
	}
	return nil
}

// resolveTarget resolves a link target reference to a canonical ULID and
// rejects a self-link (an item cannot be its own parent or blocker).
func resolveTarget(resolver *id.Resolver, it *item.Item, edge, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("%s: a target reference is required", edge)
	}
	ulid, err := resolver.Resolve(ref)
	if err != nil {
		return "", fmt.Errorf("%s: %v", edge, err)
	}
	if ulid == it.ID {
		return "", fmt.Errorf("%s: an item cannot link to itself", edge)
	}
	return ulid, nil
}

func edgeLabel(t LinkTarget) string {
	if t == LinkEpic {
		return "epic"
	}
	return "blocked-by"
}
