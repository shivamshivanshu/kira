package core

import (
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

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

func (s *Store) mutate(cfg *datamodel.Config, ref string, force bool, apply applyFn, subjectOf func(orig *datamodel.Item) string) (*datamodel.Item, []string, error) {
	release, orig, items, resolver, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, nil, err
	}
	defer release()

	updated := cloneItem(orig)
	hard, warns := apply(updated, resolver, items)
	if len(hard) > 0 {
		return nil, nil, errx.Invalid(hard)
	}
	vhard, vwarns := validateAssembled(cfg, updated, resolver, force)
	if len(vhard) == 0 {
		vhard = validateGraph(updated, items)
	}
	if len(vhard) > 0 {
		return nil, nil, errx.Invalid(vhard)
	}
	warns = append(warns, vwarns...)

	changed := datamodel.ChangedFields(orig, updated)
	if err := s.commitMutation(cfg, updated, changed, warns, subjectOf(orig)); err != nil {
		return nil, nil, err
	}
	return updated, changed, nil
}

func (s *Store) commitMutation(cfg *datamodel.Config, updated *datamodel.Item, changed []string, warns []error, subject string) error {
	if len(changed) == 0 {
		return nil
	}
	updated.Updated = time.Now().Format(time.RFC3339)

	emitWarnings(warns)

	path, err := s.writeItem(updated)
	if err != nil {
		return err
	}
	return s.finalize(cfg.Commit.Mode, cfg.Commit.Trailer, subject, updated.Number, path)
}
