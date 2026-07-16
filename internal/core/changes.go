package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func (s *Store) Changes(ref string) (*datamodel.ChangesResult, error) {
	repo := s.repo()
	sinceSHA, err := repo.ResolveTreeish(ref)
	if err != nil {
		return nil, errx.User("resolving %s: %v", ref, err)
	}
	headSHA, err := repo.ResolveTreeish("HEAD")
	if err != nil {
		return nil, errx.User("resolving HEAD: %v", err)
	}
	from, err := treeish.Load(repo, sinceSHA)
	if err != nil {
		return nil, err
	}
	to, err := treeish.Load(repo, headSHA)
	if err != nil {
		return nil, err
	}
	fromByID := byULID(from.Items)
	toByID := byULID(to.Items)

	events, err := s.rangeEvents(repo, sinceSHA, headSHA, toByID)
	if err != nil {
		return nil, err
	}

	items := changedItems(repo, fromByID, toByID, events)
	sortByKey(items, func(c datamodel.ChangedItem) id.SortKey { return id.NewSortKey(c.Number, c.ID) })
	return &datamodel.ChangesResult{Since: sinceSHA, Head: headSHA, Items: items, StderrNotes: mergedWarnings(from, to)}, nil
}

func changedItems(repo gitx.Repo, fromByID, toByID map[string]*datamodel.Item, events map[string][]datamodel.Event) []datamodel.ChangedItem {
	items := []datamodel.ChangedItem{}
	for ulid, t := range toByID {
		f, existed := fromByID[ulid]
		if !existed {
			items = append(items, newChangedItem(t, datamodel.DiffCreated, events[ulid], nil))
			continue
		}
		body := bodyDelta(repo, f, t)
		if evs := events[ulid]; len(evs) > 0 || body != nil {
			items = append(items, newChangedItem(t, datamodel.DiffChanged, evs, body))
		}
	}
	for ulid, f := range fromByID {
		if _, present := toByID[ulid]; !present {
			items = append(items, newChangedItem(f, datamodel.DiffDeleted, nil, nil))
		}
	}
	return items
}

func bodyDelta(repo gitx.Repo, from, to *datamodel.Item) *datamodel.BodyDelta {
	if from.Body == to.Body {
		return nil
	}
	added, removed, _ := repo.NumstatNoIndex(from.Body, to.Body)
	if added == 0 && removed == 0 {
		return nil
	}
	return &datamodel.BodyDelta{Added: added, Removed: removed}
}

func newChangedItem(it *datamodel.Item, status datamodel.DiffStatus, events []datamodel.Event, body *datamodel.BodyDelta) datamodel.ChangedItem {
	if events == nil {
		events = []datamodel.Event{}
	}
	return datamodel.ChangedItem{ID: it.ID, Number: it.Number, Title: it.Title, Status: status, Body: body, Events: events}
}
