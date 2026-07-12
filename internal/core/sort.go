package core

import (
	"sort"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
	"github.com/shivamshivanshu/kira/internal/query"
)

// Decorates each element with its key once so the sort never recomputes keys
// per comparison (unlike slices.SortStableFunc).
func sortByKey[T any, K interface{ Less(K) bool }](xs []T, key func(T) K) {
	type dec struct {
		k K
		v T
	}
	ds := make([]dec, len(xs))
	for i, x := range xs {
		ds[i] = dec{key(x), x}
	}
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].k.Less(ds[j].k) })
	for i := range ds {
		xs[i] = ds[i].v
	}
}

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

func precedenceKeyOf(priorityIndex map[string]int, it *item.Item) precedenceKey {
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

func sortByPrecedence(cfg *config.Config, items []*item.Item) {
	priorityIndex := query.PriorityIndex(cfg.Priorities)
	sortByKey(items, func(it *item.Item) precedenceKey { return precedenceKeyOf(priorityIndex, it) })
}
