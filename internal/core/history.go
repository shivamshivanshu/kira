package core

import (
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/ptr"
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

func (s *Store) cachedEvents(ulid, fileHead string) (events []datamodel.Event, committed bool, err error) {
	events, _, err = index.LogEntries(s.fs(), ulid, fileHead, func() ([]datamodel.Event, error) {
		return s.deriveEvents(ulid)
	})
	return events, fileHead != "", err
}

func (s *Store) cachedStateEvents(ulid, fileHead string) (events []stateTransition, committed bool, err error) {
	all, committed, err := s.cachedEvents(ulid, fileHead)
	if err != nil {
		return nil, false, err
	}
	for _, e := range all {
		if e.Field != datamodel.KeyState {
			continue
		}
		ts, perr := time.Parse(time.RFC3339, e.Ts)
		if perr != nil {
			continue
		}
		events = append(events, stateTransition{ts: ts, from: e.Old, to: e.New})
	}
	slices.Reverse(events)
	return events, committed, nil
}

type metricItem struct {
	number    string
	created   time.Time
	doingAt   time.Time
	doneAt    time.Time
	hasDoing  bool
	hasDone   bool
	doneDay   string
	degraded  bool
	dropped   bool
	category  datamodel.Category
	estimate  float64
	estimated bool
	reopens   int
}

func (s *Store) itemMetrics(cfg *datamodel.Config, it *datamodel.Item, fileHead string) (metricItem, error) {
	evs, committed, err := s.cachedStateEvents(it.ID, fileHead)
	if err != nil {
		return metricItem{}, err
	}
	mi := metricItem{
		number:    it.Number,
		estimate:  ptr.Deref(it.Estimate),
		estimated: it.Estimate != nil,
		dropped:   isDropped(it),
	}
	mi.category, _ = categoryOf(cfg, it.Type, it.State)
	if c, cerr := it.CreatedTime(); cerr == nil {
		mi.created = c
	}
	wf, hasWorkflow := cfg.Workflows[it.Type]
	doneSeen := false
	for _, ev := range evs {
		cat, _ := categoryOf(cfg, it.Type, ev.to)
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
				mi.doneDay = ev.ts.Local().Format(time.DateOnly)
			}
		}
		if ev.from != "" && hasWorkflow && wf.EnforceTransitions && !transitionAllowed(wf, ev.from, ev.to) {
			mi.degraded = true
		}
	}
	if !mi.hasDone && isDoneState(cfg, it.Type, it.State) {
		switch {
		case committed && !mi.created.IsZero():
			mi.hasDone = true
			mi.doneAt = mi.created
			mi.doneDay = mi.created.Local().Format(time.DateOnly)
		default:
			if updated, uerr := it.UpdatedTime(); uerr == nil {
				mi.hasDone = true
				mi.doneAt = updated
				mi.doneDay = updated.Local().Format(time.DateOnly)
				mi.degraded = true
			}
		}
	}
	return mi, nil
}
