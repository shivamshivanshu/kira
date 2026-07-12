package core

import (
	"time"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// MutationResult is the --json shape shared by the single-field mutations
// (edit, assign, link). Changed lists the mutable fields the command actually
// altered, empty for a no-op. These shapes are not pinned by the doc set; this
// is the chosen contract.
type MutationResult struct {
	ID      string   `json:"id"`
	Number  string   `json:"number"`
	Changed []string `json:"changed"`
}

// applyFn mutates a clone of the resolved item in place, returning hard errors
// (which block the write) and warnings (surfaced, non-blocking). resolver is
// passed for mutations that resolve a reference argument (link); items is the
// full scan the resolution came from, for mutations that need a store-wide
// census (move's WIP check).
type applyFn func(it *item.Item, resolver *id.Resolver, items []*item.Item) (hard, warns []error)

// lockAndResolve takes the store lock and resolves ref to its item, plus the
// full scan and the resolver built from it — the shared opening of every
// single-item mutation (mutate, edit, comment). The caller must defer the
// returned release.
func (s *Store) lockAndResolve(cfg *config.Config, ref string) (func(), *item.Item, []*item.Item, *id.Resolver, error) {
	release, err := s.lock()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	orig, items, resolver, err := s.resolveRef(cfg, ref)
	if err != nil {
		release()
		return nil, nil, nil, nil, err
	}
	return release, orig, items, resolver, nil
}

// mutate is the shared single-item mutation pipeline (KIRA-1): lock, resolve
// ref, clone, apply the caller's change, validate the result, then write and
// commit. It is the one place move/assign/link route through, so
// lock/validate/commit behavior cannot drift between them. subjectOf builds the
// commit subject from the resolved (pre-mutation) item. comment does not use
// this — it is a pure byte-suffix append that must not reserialize frontmatter
// (docs/design/02-data-model.md §4).
func (s *Store) mutate(cfg *config.Config, ref string, force bool, apply applyFn, subjectOf func(orig *item.Item) string) (*item.Item, []string, error) {
	release, orig, items, resolver, err := s.lockAndResolve(cfg, ref)
	if err != nil {
		return nil, nil, err
	}
	defer release()

	updated := cloneItem(orig)
	hard, warns := apply(updated, resolver, items)
	if len(hard) > 0 {
		return nil, nil, invalidErr(hard)
	}
	vhard, vwarns := validateAssembled(cfg, updated, resolver, force)
	if len(vhard) > 0 {
		return nil, nil, invalidErr(vhard)
	}
	warns = append(warns, vwarns...)

	changed := changedFields(orig, updated)
	if err := s.commitMutation(cfg, updated, changed, warns, subjectOf(orig)); err != nil {
		return nil, nil, err
	}
	return updated, changed, nil
}

// commitMutation is the shared write-and-commit tail of every frontmatter
// mutation (edit and the mutate pipeline): when changed is non-empty, bump
// updated, surface warnings, write the file, and commit under subject. An empty
// changed is a no-op — nothing is written or committed. The caller computes
// changed (and, for edit, folds it into subject).
func (s *Store) commitMutation(cfg *config.Config, updated *item.Item, changed []string, warns []error, subject string) error {
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
