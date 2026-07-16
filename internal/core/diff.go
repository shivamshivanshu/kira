package core

import (
	"fmt"
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func (s *Store) Diff(ref, since string, incoming bool) (*datamodel.DiffResult, error) {
	repo := s.repo()
	baseSHA, toSHA, err := diffEndpoints(repo, ref, since, incoming)
	if err != nil {
		return nil, err
	}
	from, err := treeish.Load(repo, baseSHA)
	if err != nil {
		return nil, err
	}
	to, err := treeish.Load(repo, toSHA)
	if err != nil {
		return nil, err
	}
	return diffSnapshots(repo, from, to, baseSHA, toSHA)
}

func diffEndpoints(repo gitx.Repo, ref, since string, incoming bool) (baseSHA, toSHA string, err error) {
	if since != "" {
		baseSHA, err = resolveDateOrTreeish(repo, since, "--since")
		if err != nil {
			return "", "", err
		}
		toSHA, err = repo.ResolveTreeish("HEAD")
		if err != nil {
			return "", "", errx.User("resolving HEAD: %v", err)
		}
		return baseSHA, toSHA, nil
	}
	target, err := repo.ResolveTreeish(ref)
	if err != nil {
		return "", "", errx.User("resolving %s: %v", ref, err)
	}
	baseSHA, err = repo.MergeBase("HEAD", target)
	if err != nil {
		return "", "", errx.User("merge-base HEAD %s: %v", ref, err)
	}
	toSHA = target
	if !incoming {
		toSHA, err = repo.ResolveTreeish("HEAD")
		if err != nil {
			return "", "", errx.User("resolving HEAD: %v", err)
		}
	}
	return baseSHA, toSHA, nil
}

func diffSnapshots(repo gitx.Repo, from, to *treeish.Loaded, fromSHA, toSHA string) (*datamodel.DiffResult, error) {
	fromByID := byULID(from.Items)
	toByID := byULID(to.Items)

	var items []datamodel.DiffItem
	for ulid, t := range toByID {
		f, ok := fromByID[ulid]
		if !ok {
			items = append(items, datamodel.DiffItem{ID: ulid, Number: t.Number, Title: t.Title, Status: datamodel.DiffCreated})
			continue
		}
		di, changed, err := changedItem(repo, f, t)
		if err != nil {
			return nil, err
		}
		if changed {
			items = append(items, di)
		}
	}
	for ulid, f := range fromByID {
		if _, ok := toByID[ulid]; !ok {
			items = append(items, datamodel.DiffItem{ID: ulid, Number: f.Number, Title: f.Title, Status: datamodel.DiffDeleted})
		}
	}

	sortDiffItems(items)
	return &datamodel.DiffResult{From: fromSHA, To: toSHA, Items: items, StderrNotes: mergedWarnings(from, to)}, nil
}

func changedItem(repo gitx.Repo, from, to *datamodel.Item) (datamodel.DiffItem, bool, error) {
	di := datamodel.DiffItem{ID: to.ID, Number: to.Number, Title: to.Title, Status: datamodel.DiffChanged}
	if from.Number != to.Number {
		if slices.Contains(to.Aliases, from.Number) {
			di.Renumbered = &datamodel.RenumberEvent{From: from.Number, To: to.Number}
		} else {
			di.Changes = append(di.Changes, datamodel.FieldChange{Field: datamodel.KeyNumber, From: from.Number, To: to.Number})
		}
	}
	for _, field := range datamodel.ChangedFields(from, to) {
		fc, err := fieldChange(repo, from, to, field)
		if err != nil {
			return datamodel.DiffItem{}, false, err
		}
		di.Changes = append(di.Changes, fc)
	}
	return di, di.Renumbered != nil || len(di.Changes) > 0, nil
}

func fieldChange(repo gitx.Repo, from, to *datamodel.Item, field string) (datamodel.FieldChange, error) {
	if field == datamodel.KeyBody {
		added, removed, err := repo.NumstatNoIndex(from.Body, to.Body)
		if err != nil {
			return datamodel.FieldChange{}, err
		}
		return datamodel.FieldChange{Field: field, To: fmt.Sprintf("+%d/-%d lines", added, removed)}, nil
	}
	return datamodel.FieldChange{Field: field, From: fieldString(from, field), To: fieldString(to, field)}, nil
}

func sortDiffItems(items []datamodel.DiffItem) {
	sortByKey(items, func(it datamodel.DiffItem) id.SortKey {
		return id.NewSortKey(it.Number, it.ID)
	})
}
