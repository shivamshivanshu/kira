package core

import (
	"fmt"
	"slices"
	"sort"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func (s *Store) Diff(ref string) (*datamodel.DiffResult, error) {
	repo := s.repo()
	target, err := repo.ResolveTreeish(ref)
	if err != nil {
		return nil, errx.User("resolving %s: %v", ref, err)
	}
	baseSha, err := repo.MergeBase("HEAD", target)
	if err != nil {
		return nil, errx.User("merge-base HEAD %s: %v", ref, err)
	}
	from, err := treeish.Load(repo, baseSha)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	to, err := treeish.Load(repo, target)
	if err != nil {
		return nil, errx.User("%v", err)
	}
	return diffSnapshots(repo, from, to, baseSha, target), nil
}

func diffSnapshots(repo gitx.Repo, from, to *treeish.Loaded, fromSha, toSha string) *datamodel.DiffResult {
	fromByID := byULID(from.Items)
	toByID := byULID(to.Items)

	var items []datamodel.DiffItem
	for ulid, t := range toByID {
		f, ok := fromByID[ulid]
		if !ok {
			items = append(items, datamodel.DiffItem{ID: ulid, Number: t.Number, Title: t.Title, Status: datamodel.DiffCreated})
			continue
		}
		if di, changed := changedItem(repo, f, t); changed {
			items = append(items, di)
		}
	}
	for ulid, f := range fromByID {
		if _, ok := toByID[ulid]; !ok {
			items = append(items, datamodel.DiffItem{ID: ulid, Number: f.Number, Title: f.Title, Status: datamodel.DiffDeleted})
		}
	}

	sortDiffItems(items)
	return &datamodel.DiffResult{From: fromSha, To: toSha, Items: items}
}

func changedItem(repo gitx.Repo, from, to *datamodel.Item) (datamodel.DiffItem, bool) {
	di := datamodel.DiffItem{ID: to.ID, Number: to.Number, Title: to.Title, Status: datamodel.DiffChanged}
	if from.Number != to.Number {
		if slices.Contains(to.Aliases, from.Number) {
			di.Renumbered = &datamodel.RenumberEvent{From: from.Number, To: to.Number}
		} else {
			di.Changes = append(di.Changes, datamodel.FieldChange{Field: datamodel.KeyNumber, From: from.Number, To: to.Number})
		}
	}
	for _, field := range datamodel.ChangedFields(from, to) {
		di.Changes = append(di.Changes, fieldChange(repo, from, to, field))
	}
	return di, di.Renumbered != nil || len(di.Changes) > 0
}

func fieldChange(repo gitx.Repo, from, to *datamodel.Item, field string) datamodel.FieldChange {
	if field == datamodel.KeyBody {
		added, removed, _ := repo.NumstatNoIndex(from.Body, to.Body)
		return datamodel.FieldChange{Field: field, To: fmt.Sprintf("+%d/-%d lines", added, removed)}
	}
	return datamodel.FieldChange{Field: field, From: fieldString(from, field), To: fieldString(to, field)}
}

func byULID(items []*datamodel.Item) map[string]*datamodel.Item {
	m := make(map[string]*datamodel.Item, len(items))
	for _, it := range items {
		m[it.ID] = it
	}
	return m
}

func sortDiffItems(items []datamodel.DiffItem) {
	keys := make(map[string]id.SortKey, len(items))
	for _, it := range items {
		keys[it.ID] = id.NewSortKey(it.Number, it.ID)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return keys[items[i].ID].Less(keys[items[j].ID])
	})
}
