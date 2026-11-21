package core

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
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

func (s *Store) fileHeads(revision string) map[string]string {
	raw, err := s.repo().LastCommits(revision, s.fs().RelToRoot(s.fs().ItemsDir()))
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
// and batching missing histories that follow the first-parent chain.
func (s *Store) itemsMetrics(cfg *datamodel.Config, items []*datamodel.Item) ([]metricItem, error) {
	if len(items) == 0 {
		return nil, nil
	}
	revision, _ := s.repo().ResolveTreeish("HEAD")
	heads := s.fileHeads(revision)
	selected := make(map[string]string, len(items))
	for _, it := range items {
		selected[it.ID] = heads[it.ID]
	}
	fields := index.StateOnly
	if cfg != nil && len(cfg.TicketStates) > 0 {
		fields = index.StateAndSubtype
	}
	byItem, err := index.StateEventsByItem(s.fs(), selected, fields, func(ids []string) (map[string][]datamodel.Event, error) {
		return s.deriveEventBatch(revision, ids)
	})
	if err != nil {
		return nil, err
	}
	var anchors map[string]*datamodel.Item
	if fields == index.StateAndSubtype {
		anchors = s.committedMetricItems(revision, items, selected)
	}
	mis := make([]metricItem, len(items))
	for i, it := range items {
		if fields == index.StateAndSubtype && it.Type == datamodel.TypeTicket {
			mis[i] = metricsFromWorkflow(cfg, it, anchors[it.ID], byItem[it.ID])
			continue
		}
		mis[i] = metricsFrom(it, stateTransitions(byItem[it.ID]), selected[it.ID] != "")
	}
	return mis, nil
}

func (s *Store) committedMetricItems(revision string, items []*datamodel.Item, heads map[string]string) map[string]*datamodel.Item {
	if revision == "" {
		return nil
	}
	var ids, specs []string
	for _, it := range items {
		if it.Type != datamodel.TypeTicket || heads[it.ID] == "" {
			continue
		}
		path := filepath.ToSlash(s.fs().RelToRoot(s.itemPath(it.ID)))
		if strings.ContainsAny(path, "\r\n") {
			continue
		}
		ids = append(ids, it.ID)
		specs = append(specs, gitx.RevPath(revision, path))
	}
	if len(specs) == 0 {
		return nil
	}
	blobs, err := s.repo().CatFileBatch(specs)
	if err != nil {
		return nil
	}
	anchors := make(map[string]*datamodel.Item, len(ids))
	for i, blob := range blobs {
		if !blob.Found {
			continue
		}
		item, err := codec.Parse(blob.Content)
		if err == nil && item.ID == ids[i] {
			anchors[item.ID] = item
		}
	}
	return anchors
}

func newMetricItem(cfg *datamodel.Config, it *datamodel.Item) metricItem {
	mi := metricItem{number: it.Number, dropped: datamodel.IsDropped(it.Resolution)}
	mi.category, _ = cfg.ItemCategory(it)
	mi.created, _ = it.CreatedTime()
	return mi
}

func (mi *metricItem) observe(category datamodel.Category, ts time.Time) {
	switch category {
	case datamodel.CategoryDoing:
		if !mi.hasDoing {
			mi.hasDoing = true
			mi.doingAt = ts
		}
		if mi.hasDone {
			mi.reopens++
		}
	case datamodel.CategoryDone:
		if !mi.hasDone {
			mi.hasDone = true
			mi.doneAt = ts
		}
	}
}

func metricsFrom(it *datamodel.Item, evs []stateTransition, committed bool) metricItem {
	mi := newMetricItem(nil, it)
	for _, ev := range evs {
		cat, _ := datamodel.CategoryOf(it.Type, ev.to)
		mi.observe(cat, ev.ts)
	}
	if !mi.hasDone && mi.category == datamodel.CategoryDone {
		switch {
		case committed && !mi.created.IsZero():
			mi.observe(datamodel.CategoryDone, mi.created)
		default:
			if updated, err := it.UpdatedTime(); err == nil {
				mi.observe(datamodel.CategoryDone, updated)
				mi.degraded = true
			}
		}
	}
	return mi
}

type categoryEvent struct {
	category datamodel.Category
	ts       time.Time
}

func metricsFromWorkflow(cfg *datamodel.Config, live, committed *datamodel.Item, events []datamodel.Event) metricItem {
	mi := newMetricItem(cfg, live)
	if committed == nil {
		mi.degraded = true
		return mi
	}
	mi.created, _ = committed.CreatedTime()

	current := *committed
	custom := workflowFor(cfg, &current).Custom
	var transitions []categoryEvent
	rooted := true
	for start := 0; start < len(events); {
		end := start + 1
		for end < len(events) && events[end].CommitSHA == events[start].CommitSHA {
			end++
		}
		previous, connected := previousMetricItem(current, events[start:end])
		if !connected {
			rooted = false
			mi.degraded = true
			break
		}
		custom = custom || workflowFor(cfg, &previous).Custom
		before, beforeKnown := cfg.ItemCategory(&previous)
		after, afterKnown := cfg.ItemCategory(&current)
		if !beforeKnown || !afterKnown {
			mi.degraded = true
		} else if before != after {
			if ts, err := time.Parse(time.RFC3339, events[start].Ts); err == nil {
				transitions = append(transitions, categoryEvent{category: after, ts: ts})
			} else {
				mi.degraded = true
			}
		}
		current, start = previous, end
	}

	if !custom && rooted {
		legacy := metricsFrom(committed, stateTransitions(events), true)
		legacy.number, legacy.category, legacy.dropped = mi.number, mi.category, mi.dropped
		return legacy
	}
	if rooted && !mi.created.IsZero() {
		category, known := cfg.ItemCategory(&current)
		mi.observe(category, mi.created)
		mi.degraded = mi.degraded || !known
	}
	for i := len(transitions) - 1; i >= 0; i-- {
		mi.observe(transitions[i].category, transitions[i].ts)
	}
	return mi
}

// The log can omit merge resolutions or a recreated ticket's initial state.
// A change whose new value disagrees with the anchor cannot identify the
// earlier state of that same ticket.
func previousMetricItem(current datamodel.Item, events []datamodel.Event) (datamodel.Item, bool) {
	previous := current
	for _, event := range events {
		switch event.Field {
		case datamodel.KeyState:
			if event.New != current.State {
				return datamodel.Item{}, false
			}
			previous.State = event.Old
		case datamodel.KeySubtype:
			if event.New != ptr.Deref(current.Subtype) {
				return datamodel.Item{}, false
			}
			previous.Subtype = ptr.NilIfEmpty(event.Old)
		}
	}
	return previous, true
}
