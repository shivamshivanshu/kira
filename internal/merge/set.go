package merge

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/setx"
)

func setMerge[T ~string](base, ours, theirs []T) []T {
	switch {
	case listEqual(ours, theirs):
		return ours
	case listEqual(ours, base):
		return theirs
	case listEqual(theirs, base):
		return ours
	}
	b, o, t := setx.ToSet(base), setx.ToSet(ours), setx.ToSet(theirs)
	var out []T
	dedup := setx.NewDeduper[T]()
	push := func(e T) {
		if dedup.Add(e) {
			out = append(out, e)
		}
	}
	for _, e := range base {
		if o[e] && t[e] {
			push(e)
		}
	}
	for _, e := range ours {
		if !b[e] {
			push(e)
		}
	}
	for _, e := range theirs {
		if !b[e] {
			push(e)
		}
	}
	return out
}

func aliasUnion(lists ...[]string) []string {
	var out []string
	dedup := setx.NewDeduper[string]()
	for _, l := range lists {
		for _, e := range l {
			if dedup.Add(e) {
				out = append(out, e)
			}
		}
	}
	return out
}

func linkMerge(base, ours, theirs map[string][]string) map[string][]string {
	out := map[string][]string{}
	for t := range keyUnion(base, ours, theirs) {
		if merged := setMerge(base[t], ours[t], theirs[t]); len(merged) > 0 {
			out[t] = merged
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func keyUnion[V any](ms ...map[string]V) map[string]bool {
	keys := map[string]bool{}
	for _, m := range ms {
		for k := range m {
			keys[k] = true
		}
	}
	return keys
}

// listEqual keeps nil distinct from empty: for extras those are different
// values (key absent vs key present and empty), so setMerge's identity fast
// paths must not collapse them the way a bare slices.Equal would.
func listEqual[T ~string](a, b []T) bool {
	return (a == nil) == (b == nil) && slices.Equal(a, b)
}
