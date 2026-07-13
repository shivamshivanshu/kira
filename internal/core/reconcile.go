package core

import (
	"fmt"
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/reconcile"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func (s *Store) Reconcile(cfg *datamodel.Config) (*datamodel.ReconcileResult, error) {
	release, err := s.fs().Lock()
	if err != nil {
		return nil, err
	}
	defer release()

	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	plan := reconcile.Plan(storage.Snapshot(cfg.Project.Key, items))

	result := &datamodel.ReconcileResult{}
	for _, r := range plan {
		it := findByULID(items, r.ULID)
		if it == nil {
			continue
		}
		if err := guardKnownFields(it); err != nil {
			return nil, err
		}
		it.Number = r.To
		if !slices.Contains(it.Aliases, r.From) {
			it.Aliases = append(it.Aliases, r.From)
		}
		path, err := s.writeItem(it)
		if err != nil {
			return nil, err
		}
		subject := fmt.Sprintf("kira: doctor renumbered %s -> %s", r.From, r.To)
		if err := s.finalize(datamodel.CommitAuto, cfg.Commit.Trailer, subject, r.To, path); err != nil {
			return nil, err
		}
		result.Renumbered = append(result.Renumbered, datamodel.Renumbering{ID: it.ID, From: r.From, To: r.To})
	}
	return result, nil
}
