package core

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// LinkTarget selects which edge kira link operates on.
type LinkTarget int

// The linkable edges. All are single-sided: only the source item's field is
// written; blocks, epic-children, and reciprocal typed-link views are
// index-derived inverses, never stored (docs/design/02-data-model.md §3).
// LinkTyped covers every links.<type> edge — the concrete type is LinkOpts.Type,
// so a new entry in item.LinkTypes needs no new enum value here.
const (
	LinkEpic LinkTarget = iota
	LinkBlockedBy
	LinkTyped
)

// LinkOpts are the resolved link inputs (docs/design/04-cli.md link). The CLI
// picks Target from whichever single edge flag was given (--epic, --blocked-by,
// or one per item.LinkTypes entry) and rejects any other count.
type LinkOpts struct {
	Target LinkTarget
	Type   string // item link type (item.LinkTypes) when Target is LinkTyped
	Ref    string // the other item's reference; may be empty only when removing an epic
	Remove bool
	Force  bool
}

// Link sets or removes one of ref's stored edges (docs/design/04-cli.md link).
// It writes only ref's own file — the inverse side (blocks, epic children) is
// derived, never stored. Self-links are rejected, and an epic may not be given
// an epic parent (epics are top-level).
func (s *Store) Link(cfg *config.Config, ref string, opts LinkOpts) (*MutationResult, error) {
	apply := func(it *item.Item, resolver *id.Resolver, _ []*item.Item) (hard, warns []error) {
		switch opts.Target {
		case LinkEpic:
			return linkEpic(it, resolver, opts), nil
		case LinkTyped:
			return linkTyped(it, resolver, opts), nil
		default:
			return linkBlockedBy(it, resolver, opts), nil
		}
	}
	subjectOf := func(orig *item.Item) string {
		verb, edge := "link", edgeLabel(opts)
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
	ulid, err := resolveTarget(resolver, "epic", opts.Ref)
	if err != nil {
		return []error{err}
	}
	it.Epic = &ulid
	return nil
}

func linkBlockedBy(it *item.Item, resolver *id.Resolver, opts LinkOpts) []error {
	ulid, err := resolveTarget(resolver, "blocked_by", opts.Ref)
	if err != nil {
		return []error{err}
	}
	it.BlockedBy = toggleRef(it.BlockedBy, ulid, opts.Remove)
	return nil
}

// linkTyped adds or removes a target in the links.<opts.Type> list, keeping the
// canonical form: a type with no targets is deleted, and a links map with no
// types is nil, so the writer omits the key (docs/design/02-data-model.md §1).
func linkTyped(it *item.Item, resolver *id.Resolver, opts LinkOpts) []error {
	linkType := opts.Type
	if !item.ValidLinkType(linkType) {
		return []error{fmt.Errorf("unknown link type %q (want one of %v)", linkType, item.LinkTypes)}
	}
	ulid, err := resolveTarget(resolver, linkType, opts.Ref)
	if err != nil {
		return []error{err}
	}
	targets := toggleRef(it.Links[linkType], ulid, opts.Remove)
	if len(targets) == 0 {
		delete(it.Links, linkType)
		if len(it.Links) == 0 {
			it.Links = nil
		}
		return nil
	}
	if it.Links == nil {
		it.Links = map[string][]string{}
	}
	it.Links[linkType] = targets
	return nil
}

// toggleRef adds ulid to (or, with remove, deletes it from) a reference list,
// idempotently — the shared edit step of every list-valued edge.
func toggleRef(list []string, ulid string, remove bool) []string {
	if remove {
		return slices.DeleteFunc(list, func(b string) bool { return b == ulid })
	}
	if slices.Contains(list, ulid) {
		return list
	}
	return append(list, ulid)
}

// resolveTarget resolves a link target reference to a canonical ULID.
// Self-link rejection is not here: it lives in normalizeRefs, which covers
// every cross-reference field on every write path, not just kira link.
func resolveTarget(resolver *id.Resolver, edge, ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("%s: a target reference is required", edge)
	}
	ulid, err := resolver.Resolve(ref)
	if err != nil {
		return "", fmt.Errorf("%s: %v", edge, err)
	}
	return ulid, nil
}

// edgeLabel names the edge for commit subjects and CLI flags: the typed-link
// flag/label form is the item link type with underscores dashed (one rule, so
// a new item.LinkTypes entry needs no mapping here).
func edgeLabel(opts LinkOpts) string {
	switch opts.Target {
	case LinkEpic:
		return "epic"
	case LinkTyped:
		return FlagForLinkType(opts.Type)
	default:
		return "blocked-by"
	}
}

// FlagForLinkType is the item link type rendered as a CLI flag name
// (duplicate_of → duplicate-of). Shared with the CLI's flag table.
func FlagForLinkType(linkType string) string {
	return strings.ReplaceAll(linkType, "_", "-")
}
