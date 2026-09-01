package core

import (
	"errors"
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
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

func (s *Store) mutate(cfg *datamodel.Config, ref string, force bool, apply applyFn, subjectOf func(orig *datamodel.Item) string, source datamodel.ChangeSource) (*datamodel.Item, []string, []error, error) {
	release, orig, items, resolver, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, nil, nil, err
	}
	defer release()
	schemas, err := s.schemas()
	if err != nil {
		return nil, nil, nil, err
	}
	vctx := newValidationCtx(cfg, items, resolver, schemas[orig.Type], force)
	return s.mutateAgainst(vctx, orig, items, apply, subjectOf, source)
}

func (s *Store) mutateAgainst(vctx *validationCtx, orig *datamodel.Item, items []*datamodel.Item, apply applyFn, subjectOf func(orig *datamodel.Item) string, source datamodel.ChangeSource) (*datamodel.Item, []string, []error, error) {
	updated := cloneItem(orig)
	hard, warns := apply(updated, vctx.resolver, items)
	if len(hard) > 0 {
		return nil, nil, nil, errx.Invalid(invalidItemPrefix, hard)
	}
	vhard, vwarns := vctx.validateMutation(orig, updated)
	if len(vhard) > 0 {
		return nil, nil, nil, errx.Invalid(invalidItemPrefix, vhard)
	}
	warns = append(warns, vwarns...)

	changed := changedKeys(orig, updated)
	finalWarns, err := s.commitMutation(vctx.cfg, orig, updated, changed, warns, subjectOf(orig), source)
	if err != nil {
		return nil, nil, nil, err
	}
	return updated, changed, finalWarns, nil
}

type Batch struct {
	cfg       *datamodel.Config
	store     *Store
	release   func()
	items     []*datamodel.Item
	resolver  *id.Resolver
	byID      map[string]*datamodel.Item
	positions map[string]int
	schemas   map[string]entityschema.Schema
	enums     map[string]map[string][]string
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
	schemas, err := s.schemas()
	if err != nil {
		release()
		return nil, err
	}
	vocab := entityschema.ConfigVocab(cfg)
	enums := make(map[string]map[string][]string, len(schemas))
	for typ, schema := range schemas {
		enums[typ] = entityschema.ResolveEnums(schema, vocab)
	}
	positions := make(map[string]int, len(ld.items))
	for n, it := range ld.items {
		if _, exists := positions[it.ID]; !exists {
			positions[it.ID] = n
		}
	}
	return &Batch{
		cfg:       cfg,
		store:     s,
		release:   release,
		items:     ld.items,
		resolver:  ld.resolver,
		byID:      datamodel.IndexByID(ld.items),
		positions: positions,
		schemas:   schemas,
		enums:     enums,
	}, nil
}

func (b *Batch) Close() { b.release() }

func (b *Batch) Resolve(ref string) (*datamodel.Item, error) {
	ulid, err := resolveID(b.resolver, ref)
	if err != nil {
		return nil, err
	}
	if n, ok := b.positions[ulid]; ok {
		return b.items[n], nil
	}
	return nil, errx.User("resolved %s to %s, which has no file", ref, ulid)
}

func (b *Batch) replace(it *datamodel.Item) {
	b.items[b.positions[it.ID]] = it
	b.byID[it.ID] = it
}

func (b *Batch) RefExists(ref string) bool {
	_, err := b.Resolve(ref)
	return err == nil
}

func (b *Batch) Mutate(ref string, force bool, apply applyFn, subjectOf func(orig *datamodel.Item) string, source datamodel.ChangeSource) (*datamodel.Item, []string, []error, error) {
	orig, err := b.Resolve(ref)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := guardWritable(orig); err != nil {
		return nil, nil, nil, err
	}
	vctx := newValidationCtxWithEnums(b.cfg, b.byID, b.resolver, b.schemas[orig.Type], b.enums[orig.Type], force)
	updated, changed, warns, err := b.store.mutateAgainst(vctx, orig, b.items, apply, subjectOf, source)
	if err != nil {
		// A write may succeed before its Git commit fails. Later operations
		// must validate against the actual on-disk state even in that case.
		if disk, readErr := storage.ReadItem(b.store.itemPath(orig.ID)); readErr == nil && disk.ID == orig.ID {
			b.replace(disk)
		}
		return nil, nil, nil, err
	}
	// The next item validated in this batch (e.g. against a WIP limit) must
	// see this commit, not the pre-batch snapshot.
	b.replace(updated)
	return updated, changed, warns, nil
}

func (s *Store) commitMutation(cfg *datamodel.Config, before, updated *datamodel.Item, changed []string, warns []error, subject string, source datamodel.ChangeSource) ([]error, error) {
	if len(changed) == 0 {
		return nil, nil
	}
	updated.Updated = time.Now().Format(time.RFC3339)
	warns = scopeVocabWarnings(warns, changed)

	path, err := s.fs().WriteItem(updated)
	if err != nil {
		return nil, err
	}
	if err := s.commit(cfg, &datamodel.ChangeSet{
		Kind:    datamodel.ChangeMutated,
		Before:  before,
		After:   updated,
		Changed: changed,
		Paths:   []string{path},
		Subject: subject,
		Source:  source,
	}); err != nil {
		return nil, err
	}
	return warns, nil
}

func scopeVocabWarnings(warns []error, changed []string) []error {
	out := warns[:0:0]
	for _, w := range warns {
		if v, ok := errors.AsType[entityschema.Violation](w); ok && !slices.Contains(changed, v.Field) {
			continue
		}
		out = append(out, w)
	}
	return out
}

func changedKeys(orig, updated *datamodel.Item) []string {
	return append(datamodel.ChangedFields(orig, updated), datamodel.ChangedExtraKeys(orig, updated)...)
}

func (s *Store) commit(cfg *datamodel.Config, cs *datamodel.ChangeSet) error {
	sha, err := s.finalize(cfg.Commit.Mode, commitSpec{trailerKey: cfg.Commit.Trailer, subject: cs.Subject, trailerNumber: cs.After.Number}, cs.Paths...)
	if err != nil {
		return err
	}
	if len(cfg.Automation) > 0 || len(cfg.UserAutomation) > 0 {
		s.fireAutomation(cfg, cs, sha)
	}
	return nil
}
