package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
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
	matches  map[string][]*datamodel.Item
}

func (s *Store) read(cfg *datamodel.Config, opts loadOpts) (*loaded, error) {
	if opts.at != "" {
		sha, err := s.resolveAtRef(opts.at)
		if err != nil {
			return nil, err
		}
		tl, err := treeish.Load(s.repo(), sha)
		if err != nil {
			return nil, err
		}
		return &loaded{items: tl.Items, resolver: tl.Resolver, cfg: tl.Config, notes: literalWarnings(tl.Warnings)}, nil
	}
	var indexErr error
	if opts.useIndex {
		items, res, err := index.Load(s.fs(), s.repo(), indexOptions(cfg))
		if err == nil {
			_, resolver := storage.SnapshotAndResolver(cfg.Project.Key, items)
			return &loaded{items: items, resolver: resolver, cfg: cfg, notes: literalWarnings(res.Warnings)}, nil
		}
		indexErr = err
	}
	ld, err := s.load(cfg)
	if err != nil {
		return nil, err
	}
	notes := literalWarnings(ld.warnings)
	if opts.useIndex {
		notes = append([]datamodel.Warning{{Code: datamodel.WarnIndexFallback, Args: []string{indexErr.Error()}}}, notes...)
	}
	return &loaded{items: ld.items, resolver: ld.resolver, cfg: cfg, notes: notes}, nil
}

func (s *Store) resolveAtRef(at string) (string, error) {
	return resolveDateOrTreeish(s.repo(), at, "--at")
}

func resolveDateOrTreeish(repo gitx.Repo, val, flag string) (string, error) {
	if datamodel.ValidDate(val) {
		sha, err := repo.ResolveAtDate(gitx.Date(val+"T00:00:00"), gitx.Ref("HEAD"))
		if err != nil {
			return "", errx.User("resolving %s %s: %v", flag, val, err)
		}
		return sha, nil
	}
	sha, err := repo.ResolveTreeish(val)
	if err != nil {
		return "", errx.User("resolving %s %s: %v", flag, val, err)
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
