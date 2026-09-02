package core

import (
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type stateTransition struct {
	ts   time.Time
	from string
	to   string
}

func (s *Store) fileHead(ulid string) string {
	head, err := s.repo().LastCommitFor(s.fs().RelToRoot(s.itemPath(ulid)))
	if err != nil {
		return ""
	}
	return head
}

func (s *Store) fileHeads() map[string]string {
	raw, err := s.repo().LastCommits(s.fs().RelToRoot(s.fs().ItemsDir()))
	if err != nil {
		return nil
	}
	heads := make(map[string]string, len(raw))
	for path, sha := range raw {
		if ulid := storage.ULIDFromPath(path); ulid != "" {
			heads[ulid] = sha
		}
	}
	return heads
}

func (s *Store) cachedStateEvents(ulid, fileHead string) (oldestFirst []stateTransition, committed bool, err error) {
	all, committed, err := s.cachedEvents(ulid, fileHead)
	if err != nil {
		return nil, false, err
	}
	return stateTransitions(all), committed, nil
}

func stateTransitions(all []datamodel.Event) (oldestFirst []stateTransition) {
	for _, e := range all {
		if e.Field != datamodel.KeyState {
			continue
		}
		ts, perr := time.Parse(time.RFC3339, e.Ts)
		if perr != nil {
			continue
		}
		oldestFirst = append(oldestFirst, stateTransition{ts: ts, from: e.Old, to: e.New})
	}
	slices.Reverse(oldestFirst)
	return oldestFirst
}

type metricItem struct {
	number   string
	created  time.Time
	doingAt  time.Time
	doneAt   time.Time
	hasDoing bool
	hasDone  bool
	degraded bool
	dropped  bool
	category datamodel.Category
	reopens  int
}

// itemsMetrics derives metrics for a whole item set, reading every item's
// cached state events through a single index open rather than one per item,
// and deriving whatever the cache is missing in a single git log.
func (s *Store) itemsMetrics(cfg *datamodel.Config, items []*datamodel.Item) ([]metricItem, error) {
	if len(items) == 0 {
		return nil, nil
	}
	heads := s.fileHeads()
	selected := make(map[string]string, len(items))
	for _, it := range items {
		selected[it.ID] = heads[it.ID]
	}
	byItem, err := index.StateEventsByItem(s.fs(), selected, s.deriveEventBatch)
	if err != nil {
		return nil, err
	}
	mis := make([]metricItem, len(items))
	for i, it := range items {
		mis[i] = metricsFrom(cfg, it, stateTransitions(byItem[it.ID]), heads[it.ID] != "")
	}
	return mis, nil
}

func metricsFrom(cfg *datamodel.Config, it *datamodel.Item, evs []stateTransition, committed bool) metricItem {
	mi := metricItem{
		number:  it.Number,
		dropped: datamodel.IsDropped(it.Resolution, cfg.ResolutionsDropped),
	}
	mi.category, _ = cfg.CategoryOf(it.Type, it.State)
	if c, cerr := it.CreatedTime(); cerr == nil {
		mi.created = c
	}
	doneSeen := false
	for _, ev := range evs {
		cat, _ := cfg.CategoryOf(it.Type, ev.to)
		if cat == datamodel.CategoryDoing {
			if !mi.hasDoing {
				mi.hasDoing = true
				mi.doingAt = ev.ts
			}
			if doneSeen {
				mi.reopens++
			}
		}
		if cat == datamodel.CategoryDone {
			doneSeen = true
			if !mi.hasDone {
				mi.hasDone = true
				mi.doneAt = ev.ts
			}
		}
	}
	if !mi.hasDone && isDoneState(cfg, it.Type, it.State) {
		switch {
		case committed && !mi.created.IsZero():
			mi.hasDone = true
			mi.doneAt = mi.created
		default:
			if updated, uerr := it.UpdatedTime(); uerr == nil {
				mi.hasDone = true
				mi.doneAt = updated
				mi.degraded = true
			}
		}
	}
	return mi
}
