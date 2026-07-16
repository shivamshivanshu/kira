package core

import (
	"errors"
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

const invalidItemPrefix = "invalid item"

type applyFn func(it *datamodel.Item, resolver *id.Resolver, items []*datamodel.Item) (hard, warns []error)

func (s *Store) lockAndResolve(cfg *datamodel.Config, ref string) (func(), *datamodel.Item, []*datamodel.Item, *id.Resolver, error) {
	release, err := s.fs().Lock()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	orig, items, resolver, err := s.resolveRef(cfg, ref)
	if err != nil {
		release()
		return nil, nil, nil, nil, err
	}
	if err := guardWritable(orig); err != nil {
		release()
		return nil, nil, nil, nil, err
	}
	return release, orig, items, resolver, nil
}

func (s *Store) mutate(cfg *datamodel.Config, ref string, force bool, apply applyFn, subjectOf func(orig *datamodel.Item) string, source datamodel.ChangeSource) (*datamodel.Item, []string, error) {
	release, orig, items, resolver, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	return s.mutateAgainst(cfg, orig, items, resolver, force, apply, subjectOf, source)
}

func (s *Store) mutateAgainst(cfg *datamodel.Config, orig *datamodel.Item, items []*datamodel.Item, resolver *id.Resolver, force bool, apply applyFn, subjectOf func(orig *datamodel.Item) string, source datamodel.ChangeSource) (*datamodel.Item, []string, error) {
	updated := cloneItem(orig)
	hard, warns := apply(updated, resolver, items)
	if len(hard) > 0 {
		return nil, nil, errx.Invalid(invalidItemPrefix, hard)
	}
	vhard, vwarns := validateMutation(cfg, orig, updated, resolver, items, force)
	if len(vhard) > 0 {
		return nil, nil, errx.Invalid(invalidItemPrefix, vhard)
	}
	warns = append(warns, vwarns...)

	changed := datamodel.ChangedFields(orig, updated)
	if err := s.commitMutation(cfg, orig, updated, changed, warns, subjectOf(orig), source); err != nil {
		return nil, nil, err
	}
	return updated, changed, nil
}

type Batch struct {
	cfg      *datamodel.Config
	store    *Store
	release  func()
	items    []*datamodel.Item
	resolver *id.Resolver
}

func (s *Store) BeginBatch(cfg *datamodel.Config) (*Batch, error) {
	release, err := s.fs().Lock()
	if err != nil {
		return nil, err
	}
	ld, err := s.load(cfg)
	if err != nil {
		release()
		return nil, err
	}
	return &Batch{cfg: cfg, store: s, release: release, items: ld.items, resolver: ld.resolver}, nil
}

func (b *Batch) Close() { b.release() }

func (b *Batch) Resolve(ref string) (*datamodel.Item, error) {
	return resolveItem(b.items, b.resolver, ref)
}

func (b *Batch) RefExists(ref string) bool {
	_, err := b.Resolve(ref)
	return err == nil
}

func (b *Batch) Mutate(ref string, force bool, apply applyFn, subjectOf func(orig *datamodel.Item) string, source datamodel.ChangeSource) (*datamodel.Item, []string, error) {
	orig, err := b.Resolve(ref)
	if err != nil {
		return nil, nil, err
	}
	if err := guardWritable(orig); err != nil {
		return nil, nil, err
	}
	updated, changed, err := b.store.mutateAgainst(b.cfg, orig, b.items, b.resolver, force, apply, subjectOf, source)
	if err != nil {
		return nil, nil, err
	}
	// The next item validated in this batch (e.g. against a WIP limit) must
	// see this commit, not the pre-batch snapshot.
	replaceByULID(b.items, updated)
	return updated, changed, nil
}

func (s *Store) commitMutation(cfg *datamodel.Config, before, updated *datamodel.Item, changed []string, warns []error, subject string, source datamodel.ChangeSource) error {
	if len(changed) == 0 {
		return nil
	}
	updated.Updated = time.Now().Format(time.RFC3339)
	warns = scopeVocabWarnings(warns, changed)

	path, err := s.fs().WriteItem(updated)
	if err != nil {
		return err
	}
	return s.commit(cfg, &datamodel.ChangeSet{
		Kind:    datamodel.ChangeMutated,
		Before:  before,
		After:   updated,
		Changed: changed,
		Paths:   []string{path},
		Subject: subject,
		Source:  source,
	}, warns)
}

func scopeVocabWarnings(warns []error, changed []string) []error {
	out := warns[:0:0]
	for _, w := range warns {
		var vw *vocabWarning
		if errors.As(w, &vw) && !slices.Contains(changed, vw.field) {
			continue
		}
		out = append(out, w)
	}
	return out
}

func (s *Store) commit(cfg *datamodel.Config, cs *datamodel.ChangeSet, warns []error) error {
	emitWarnings(warns)
	sha, err := s.finalize(cfg.Commit.Mode, commitSpec{trailerKey: cfg.Commit.Trailer, subject: cs.Subject, trailerNumber: cs.After.Number}, cs.Paths...)
	if err != nil {
		return err
	}
	if len(cfg.Automation) > 0 || len(cfg.UserAutomation) > 0 {
		s.fireAutomation(cfg, cs, sha)
	}
	return nil
}
