package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
)

type precedenceKey struct {
	rank          string
	priorityIndex int
	number        id.SortKey
}

func (k precedenceKey) Less(o precedenceKey) bool {
	if (k.rank == "") != (o.rank == "") {
		return k.rank != ""
	}
	if k.rank != o.rank {
		return k.rank < o.rank
	}
	if k.priorityIndex != o.priorityIndex {
		return k.priorityIndex < o.priorityIndex
	}
	return k.number.Less(o.number)
}

func precedenceKeyOf(priorityIndex map[string]int, it *datamodel.Item) precedenceKey {
	k := precedenceKey{priorityIndex: len(priorityIndex), number: id.NewSortKey(it.Number, it.ID)}
	if it.Rank != nil {
		k.rank = *it.Rank
	}
	if it.Priority != nil {
		if i, ok := priorityIndex[*it.Priority]; ok {
			k.priorityIndex = i
		}
	}
	return k
}

func sortByPrecedence(cfg *datamodel.Config, items []*datamodel.Item) {
	priorityIndex := query.PriorityIndex(cfg.Priorities.Values)
	sortByKey(items, func(it *datamodel.Item) precedenceKey { return precedenceKeyOf(priorityIndex, it) })
}
