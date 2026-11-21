package core

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
)

func (s *Store) Blockers(cfg *datamodel.Config, ref string) (*datamodel.BlockersResult, error) {
	ld, err := s.read(cfg, loadOpts{})
	if err != nil {
		return nil, err
	}
	root, err := findItem(ld.items, ld.resolver, ref)
	if err != nil {
		return nil, err
	}
	result, err := blockerReport(ld.cfg, root, ld.items)
	if err != nil {
		return nil, err
	}
	result.StderrNotes = ld.notes
	return result, nil
}

func blockerReport(cfg *datamodel.Config, root *datamodel.Item, items []*datamodel.Item) (*datamodel.BlockersResult, error) {
	isEpic := datamodel.IsContainer(root.Type)
	var byID map[string]*datamodel.Item
	var byParent map[string][]*datamodel.Item
	if isEpic {
		byID, byParent = indexByParent(items)
	} else {
		byID = datamodel.IndexByID(items)
	}
	leaves, incomplete := searchBlockers(cfg, root, byID)
	result := &datamodel.BlockersResult{
		Item:       blockerItem(root),
		Actionable: []datamodel.ActionableBlocker{},
		Direct:     []datamodel.ActionableBlocker{},
		Incomplete: incomplete,
	}
	if !isEpic {
		result.Actionable = leaves
		sortBlockerResults(result.Actionable)
		return result, nil
	}
	for idx := range leaves {
		leaves[idx].Affected = 0
	}
	result.Direct = leaves
	sortBlockerResults(result.Direct)

	children, err := descendants(byParent, root.ID)
	if err != nil {
		return nil, err
	}
	sortItems(children)
	leafPositions := map[string]int{}
	for _, child := range children {
		if child.Type != datamodel.TypeTicket {
			continue
		}
		found, unresolved := searchBlockers(cfg, child, byID)
		result.Incomplete = append(result.Incomplete, unresolved...)
		for _, leaf := range found {
			if position, ok := leafPositions[leaf.Item.ID]; ok {
				result.Actionable[position].Affected++
				continue
			}
			leafPositions[leaf.Item.ID] = len(result.Actionable)
			result.Actionable = append(result.Actionable, leaf)
		}
	}
	sortBlockerResults(result.Actionable)
	return result, nil
}

func blockerItem(it *datamodel.Item) datamodel.BlockerItem {
	return datamodel.BlockerItem{ID: it.ID, Number: numberOrID(it, it.ID), Title: it.Title, State: it.State}
}

type blockerSearch struct {
	cfg        *datamodel.Config
	byID       map[string]*datamodel.Item
	visits     map[string]blockerVisit
	leaves     []datamodel.ActionableBlocker
	incomplete []datamodel.BlockerIssue
}

type blockerVisit uint8

const (
	blockerUnseen blockerVisit = iota
	blockerOnPath
	blockerComplete
)

func searchBlockers(cfg *datamodel.Config, root *datamodel.Item, byID map[string]*datamodel.Item) ([]datamodel.ActionableBlocker, []datamodel.BlockerIssue) {
	search := blockerSearch{
		cfg:        cfg,
		byID:       byID,
		visits:     map[string]blockerVisit{},
		leaves:     []datamodel.ActionableBlocker{},
		incomplete: []datamodel.BlockerIssue{},
	}
	category, known := cfg.ItemCategory(root)
	switch {
	case category == datamodel.CategoryDone || datamodel.IsDropped(root.Resolution):
	case !known:
		search.issue(nil, numberOrID(root, root.ID), "has no known state category")
	case len(root.BlockedBy) > 0:
		search.walk(root, nil)
	}
	return search.leaves, search.incomplete
}

func (s *blockerSearch) walk(node *datamodel.Item, path []string) {
	s.visits[node.ID] = blockerOnPath
	path = append(path, numberOrID(node, node.ID))
	open, skipped := query.OpenBlockers(s.cfg, node, s.byID, query.SkipUnknownBlockers)
	hasBlocker := false
	for _, unresolved := range skipped {
		if unresolved.Blocker != nil && datamodel.IsDropped(unresolved.Blocker.Resolution) {
			continue
		}
		hasBlocker = true
		s.issue(path, numberOrID(unresolved.Blocker, unresolved.ID), unresolved.Reason)
	}
	for _, blocker := range open {
		if datamodel.IsDropped(blocker.Resolution) {
			continue
		}
		hasBlocker = true
		switch s.visits[blocker.ID] {
		case blockerOnPath:
			s.issue(path, numberOrID(blocker, blocker.ID), "blocker cycle")
		case blockerUnseen:
			s.walk(blocker, path)
		}
	}
	if !hasBlocker && len(path) > 1 {
		s.leaves = append(s.leaves, datamodel.ActionableBlocker{Item: blockerItem(node), Affected: 1, Path: slices.Clone(path)})
	}
	s.visits[node.ID] = blockerComplete
}

func (s *blockerSearch) issue(path []string, ref, reason string) {
	s.incomplete = append(s.incomplete, datamodel.BlockerIssue{Path: append(slices.Clone(path), ref), Reason: reason})
}

type blockerRank struct {
	affected int
	item     id.SortKey
}

func (r blockerRank) Less(other blockerRank) bool {
	if r.affected != other.affected {
		return r.affected > other.affected
	}
	return r.item.Less(other.item)
}

func sortBlockerResults(leaves []datamodel.ActionableBlocker) {
	sortByKey(leaves, func(leaf datamodel.ActionableBlocker) blockerRank {
		return blockerRank{affected: leaf.Affected, item: id.NewSortKey(leaf.Item.Number, leaf.Item.ID)}
	})
}
