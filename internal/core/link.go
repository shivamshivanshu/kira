package core

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

type LinkTarget int

const (
	LinkEpic LinkTarget = iota
	LinkBlockedBy
	LinkTyped
)

type LinkOpts struct {
	Target LinkTarget
	Type   string
	Ref    string
	Remove bool
	Force  bool
}

func (s *Store) Link(cfg *datamodel.Config, ref string, opts LinkOpts) (*datamodel.MutationResult, error) {
	apply := func(it *datamodel.Item, resolver *id.Resolver, _ []*datamodel.Item) (hard, warns []error) {
		switch opts.Target {
		case LinkEpic:
			return linkEpic(it, resolver, opts), nil
		case LinkTyped:
			return linkTyped(it, resolver, opts), nil
		default:
			return linkBlockedBy(it, resolver, opts), nil
		}
	}
	subjectOf := func(orig *datamodel.Item) string {
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
	return &datamodel.MutationResult{ID: updated.ID, Number: updated.Number, Changed: changed}, nil
}

func linkEpic(it *datamodel.Item, resolver *id.Resolver, opts LinkOpts) []error {
	if opts.Remove {
		it.Epic = nil
		return nil
	}
	if it.Type == datamodel.TypeEpic {
		return []error{fmt.Errorf("an epic cannot have an epic parent")}
	}
	ulid, err := resolveTarget(resolver, "epic", opts.Ref)
	if err != nil {
		return []error{err}
	}
	it.Epic = &ulid
	return nil
}

func linkBlockedBy(it *datamodel.Item, resolver *id.Resolver, opts LinkOpts) []error {
	ulid, err := resolveTarget(resolver, "blocked_by", opts.Ref)
	if err != nil {
		return []error{err}
	}
	it.BlockedBy = toggleRef(it.BlockedBy, ulid, opts.Remove)
	return nil
}

func linkTyped(it *datamodel.Item, resolver *id.Resolver, opts LinkOpts) []error {
	linkType := opts.Type
	if !datamodel.ValidLinkType(linkType) {
		return []error{fmt.Errorf("unknown link type %q (want one of %v)", linkType, datamodel.LinkTypes)}
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

func toggleRef(list []string, ulid string, remove bool) []string {
	if remove {
		return slices.DeleteFunc(list, func(b string) bool { return b == ulid })
	}
	if slices.Contains(list, ulid) {
		return list
	}
	return append(list, ulid)
}

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

func FlagForLinkType(linkType string) string {
	return strings.ReplaceAll(linkType, "_", "-")
}
