package core

import (
	"sort"

	"github.com/shivamshivanshu/kira/internal/id"
)

// sortByKey stably sorts xs by the id.SortKey each element maps to. It decorates
// every element with its key in one O(n) pass and sorts the decorated slice, so
// the key is computed once per element — unlike slices.SortStableFunc, which
// would recompute it on every comparison (and re-hit id.ParseNumber's error
// path for hash-style numbers).
func sortByKey[T any](xs []T, key func(T) id.SortKey) {
	type dec struct {
		k id.SortKey
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
