package core

import "sort"

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
