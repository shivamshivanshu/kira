package core

import "slices"

func sortByKey[T any, K interface{ Less(K) bool }](xs []T, key func(T) K) {
	type dec struct {
		k K
		v T
	}
	ds := make([]dec, len(xs))
	for i, x := range xs {
		ds[i] = dec{key(x), x}
	}
	slices.SortStableFunc(ds, func(a, b dec) int {
		switch {
		case a.k.Less(b.k):
			return -1
		case b.k.Less(a.k):
			return 1
		default:
			return 0
		}
	})
	for i := range ds {
		xs[i] = ds[i].v
	}
}
