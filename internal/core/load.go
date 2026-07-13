package core

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

type loadOpts struct {
	at       string
	useIndex bool
}

type loaded struct {
	items    []*datamodel.Item
	resolver *id.Resolver
	cfg      *datamodel.Config
	notes    []datamodel.Warning
}

func (s *Store) read(cfg *datamodel.Config, opts loadOpts) (*loaded, error) {
	ld, err := s.readRaw(cfg, opts)
	if err != nil {
		return nil, err
	}
	ld.notes = append(ld.notes, orphanTypeNotes(ld.cfg, ld.items)...)
	return ld, nil
}

func orphanTypeNotes(cfg *datamodel.Config, items []*datamodel.Item) []datamodel.Warning {
	var out []datamodel.Warning
	for _, it := range items {
		if _, ok := cfg.Workflows[it.Type]; !ok {
			out = append(out, datamodel.Warning{Code: datamodel.WarnOrphanType, Args: []string{it.Number, it.Type}})
		}
	}
	slices.SortFunc(out, func(a, b datamodel.Warning) int { return strings.Compare(a.Args[0], b.Args[0]) })
	return out
}

func (s *Store) readRaw(cfg *datamodel.Config, opts loadOpts) (*loaded, error) {
	if opts.at != "" {
		sha, err := s.resolveAtRef(opts.at)
		if err != nil {
			return nil, err
		}
		tl, err := treeish.Load(s.repo(), sha)
		if err != nil {
			return nil, errx.User("%v", err)
		}
		return &loaded{items: tl.Items, resolver: tl.Resolver, cfg: tl.Config}, nil
	}
	if opts.useIndex {
		if items, res, err := index.Load(s.fs(), s.repo(), indexOptions(cfg)); err == nil {
			_, resolver := snapshotAndResolver(cfg.Project.Key, items)
			return &loaded{items: items, resolver: resolver, cfg: cfg, notes: literalWarnings(res.Warnings)}, nil
		}
	}
	items, _, resolver, warnings, err := s.load(cfg)
	if err != nil {
		return nil, err
	}
	notes := literalWarnings(warnings)
	if opts.useIndex {
		notes = append([]datamodel.Warning{{Code: datamodel.WarnIndexFallback}}, notes...)
	}
	return &loaded{items: items, resolver: resolver, cfg: cfg, notes: notes}, nil
}

func (s *Store) resolveAtRef(at string) (string, error) {
	repo := s.repo()
	if datamodel.ValidDate(at) {
		sha, err := repo.ResolveAtDate(at, "HEAD")
		if err != nil {
			return "", errx.User("resolving --at %s: %v", at, err)
		}
		return sha, nil
	}
	sha, err := repo.ResolveTreeish(at)
	if err != nil {
		return "", errx.User("resolving --at %s: %v", at, err)
	}
	return sha, nil
}

func (s *Store) skew(cfg *datamodel.Config, ref, atULID, at string) *datamodel.Skew {
	ld, err := s.read(cfg, loadOpts{})
	if err != nil {
		return nil
	}
	nowULID, err := ld.resolver.Resolve(ref)
	if err != nil || nowULID == atULID {
		return nil
	}
	return &datamodel.Skew{Ref: ref, At: at, AtID: atULID, NowID: nowULID}
}
